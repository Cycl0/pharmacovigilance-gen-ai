from unsloth import FastLanguageModel
import torch
from peft import PeftModel

MODEL_ID = "Qwen/Qwen2.5-7B-Instruct"
LORA_PATH = "output/lora_adapters"
OUTPUT_DATA_PATH = "output/merged_model"

model, tokenizer = FastLanguageModel.from_pretrained(
    model_name=MODEL_ID,
    max_seq_length=2048,
    dtype=torch.bfloat16,  # Use bfloat16 for better numerical stability
    load_in_4bit=False,
    load_in_8bit=False,
)

# Load the LoRA adapters
model = PeftModel.from_pretrained(model, LORA_PATH)

# Merge the adapters with the base model
merged_model = model.merge_and_unload()

# Save the fully merged model
merged_model.save_pretrained(
    OUTPUT_DATA_PATH,
    safe_serialization=True,
    max_shard_size="5GB"
)

tokenizer.save_pretrained(OUTPUT_DATA_PATH)



