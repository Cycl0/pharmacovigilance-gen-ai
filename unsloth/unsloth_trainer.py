# Import Unsloth first to ensure all optimizations are applied
from unsloth import FastLanguageModel
import torch
from transformers import AutoTokenizer, TrainingArguments, AutoModelForCausalLM, AutoConfig
import torch
from trl import SFTTrainer
from datasets import load_dataset, load_from_disk
from peft import LoraConfig, get_peft_model
import os
from trl import DataCollatorForCompletionOnlyLM

MODEL_ID = "deepcogito/cogito-v1-preview-llama-3B"
TRAINING_DATA_PATH = "training_data_set"
OUTPUT_DATA_PATH = "output"
NUM_EPOCHS = 5
USE_QLORA = False

model, tokenizer = FastLanguageModel.from_pretrained(
    model_name=MODEL_ID,
    max_seq_length=2048,
    dtype=torch.bfloat16,  # Use bfloat16 for better numerical stability
    load_in_4bit=USE_QLORA,
    load_in_8bit=False,
)

# Apply LoRA to the model
if USE_QLORA:
    # LoRA configuration optimized for maximum quality
    lora_config = LoraConfig(
        r=64,                     # Higher rank for better quality
        lora_alpha=64,           # Alpha scaling for LoRA
        target_modules=["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"],
        lora_dropout=0.05,       # Slight dropout for regularization
        bias="none",
        task_type="CAUSAL_LM",
        modules_to_save=["embed_tokens", "lm_head"]  # Save these layers in full precision
    )
    model = get_peft_model(model, lora_config)

# Define formatting function for Qwen2.5 chat format
def format_prompts_func(example):
    # Convert dataset format to Qwen2.5 chat format
    if "input" in example and example["input"]:
        user_content = f"{example['instruction']}\n{example['input']}"
    else:
        user_content = example["instruction"]

    formatted_text = f"<|im_start|>user\n{user_content}<|im_end|>\n<|im_start|>assistant\n{example['output']}<|im_end|>\n"
    return formatted_text

# Set up data collator for completion-only fine-tuning
response_template = "<|im_start|>assistant\n"
data_collator = DataCollatorForCompletionOnlyLM(
    response_template=response_template,
    tokenizer=tokenizer,
    mlm=False
)

# Load and prepare dataset - using the correct path
try:
    # First try loading from disk
    dataset = load_from_disk(TRAINING_DATA_PATH)
    print(f"Successfully loaded dataset from {TRAINING_DATA_PATH}")
except Exception as e:
    print(f"Error loading dataset from {TRAINING_DATA_PATH}: {e}")
    print("Trying to create a new dataset...")

    # If loading fails, create a new dataset
    from create_dataset import create_dataset
    create_dataset()

    # Try loading again
    dataset = load_from_disk(TRAINING_DATA_PATH)
    print(f"Successfully loaded newly created dataset from {TRAINING_DATA_PATH}")

# Format function for the dataset
def format_instruction(example):
    return {
        "text": f"<|im_start|>system\nYou are a helpful AI assistant specialized in pharmacovigilance.\n<|im_end|>\n<|im_start|>user\n{example['input']}\n<|im_end|>\n<|im_start|>assistant\n{example['output']}\n<|im_end|>"
    }

# Apply formatting
formatted_dataset = dataset.map(format_instruction)

# Training arguments optimized for maximum quality
training_args = TrainingArguments(
    output_dir=OUTPUT_DATA_PATH,
    num_train_epochs=NUM_EPOCHS,    # Increased epochs for better learning
    per_device_train_batch_size=16 if USE_QLORA else 16,
    gradient_accumulation_steps=2 if USE_QLORA else 1,
    learning_rate=1e-4,             # Scaled LR for larger batches [[16]]
    fp16=False,                     # Disable fp16 for better precision
    bf16=True,                      # Use bfloat16 instead
    logging_steps=10,
    save_strategy="steps",          # Save more frequently
    save_steps=500,                 # Save every 500 steps
    warmup_steps=500,               # Longer warmup
    weight_decay=0.01,              # Increased weight decay for better regularization
    lr_scheduler_type="cosine",     # Cosine learning rate schedule
    save_total_limit=2,             # Keep more checkpoints
    gradient_checkpointing=USE_QLORA,    # Enable gradient checkpointing for memory efficiency
    optim="adamw_torch",            # Use PyTorch's AdamW implementation
    max_grad_norm=1.0,              # Gradient clipping
    # Explicitly disable DeepSpeed
    deepspeed=None,
)

# Initialize trainer
trainer = SFTTrainer(
    model=model,
    train_dataset=formatted_dataset,
    args=training_args,
    tokenizer=tokenizer,
    max_seq_length=2048,
)

# Train the model
trainer.train()

if hasattr(model, "_compiled_call_impl"):
    model._compiled_call_impl = None

if USE_QLORA:
    print("Saving adapters...")
    # Save ONLY THE ADAPTERS (no merging)
    # This saves:
    # - adapter_config.json
    # - adapter_model.safetensors
    model.save_pretrained(os.path.join(OUTPUT_DATA_PATH, "qlora_adapters"))
    tokenizer.save_pretrained(os.path.join(OUTPUT_DATA_PATH, "qlora_adapters"))
    print("Saved adapter files:", os.path.join(OUTPUT_DATA_PATH, "qlora_adapters"))
else:
    print("Saving full model")
    trainer.model.save_pretrained(os.path.join(OUTPUT_DATA_PATH, "full_model"))
    trainer.tokenizer.save_pretrained(os.path.join(OUTPUT_DATA_PATH,"full_model"))
    print("Saved full model files:", os.path.join(OUTPUT_DATA_PATH, "full_model"))
print("Saving tokenizer...")
