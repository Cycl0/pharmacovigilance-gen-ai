# Use NVIDIA CUDA development image
FROM nvidia/cuda:12.3.1-devel-ubuntu22.04

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive
ENV TORCH_HOME=/root/.cache/torch
ENV HF_HOME=/model_cache
ENV LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH
ENV PATH=/usr/local/cuda/bin:$PATH
ENV CUDA_HOME=/usr/local/cuda
ENV CUDA_VISIBLE_DEVICES=0
ENV CUDA_VERSION=12.1
ENV BNB_CUDA_VERSION=121
ENV TORCH_CUDA_ARCH_LIST="8.6"
# Add memory management settings
ENV PYTORCH_CUDA_ALLOC_CONF=max_split_size_mb:128
ENV OMP_NUM_THREADS=4
ENV TOKENIZERS_PARALLELISM=false
ENV PYTORCH_NO_CUDA_MEMORY_CACHING=1
ENV CUDA_LAUNCH_BLOCKING=1
ENV TORCH_USE_CUDA_DSA=1
ENV PYTORCH_ENABLE_MPS_FALLBACK=1
ENV ACCELERATE_MIXED_PRECISION=bf16
ENV ACCELERATE_LOAD_BALANCING=1
ENV ACCELERATE_CPU_OFFLOAD=1

# Install system dependencies
RUN apt-get update && apt-get install -y \
    python3.10 \
    python3-pip \
    git \
    cmake \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Create necessary directories
RUN mkdir -p /root/.cache/torch /trainer/training_data_set /trainer/output /model_cache

# Install PyTorch with specific CUDA version
RUN pip3 install --no-cache-dir torch==2.1.0+cu121 -f https://download.pytorch.org/whl/cu121/torch_stable.html

# Install NumPy 1.x first to avoid compatibility issues
RUN pip3 install --no-cache-dir "numpy<2.0"

# Install other Python packages
RUN pip3 install --no-cache-dir \
    transformers==4.46.0 \
    accelerate==0.34.0 \
    peft==0.15.1 \
    trl==0.12.0 \
    datasets==2.21.0 \
    sentencepiece==0.1.99 \
    protobuf==3.20.3 \
    scipy==1.11.4
    
# Install bitsandbytes with CUDA 12.1 support
RUN pip3 install --no-cache-dir bitsandbytes==0.45.4

# Install unsloth with Qwen support
RUN pip3 install --no-cache-dir unsloth==2025.3.19

# Copy training script and dataset
COPY unsloth_trainer.py convert_to_gguf.py create_dataset.py merge_with_full_model.py /trainer/

# Set working directory
WORKDIR /trainer

# Run the training script, create dataset, and convert to GGUF

# QLoRA/LoRA
CMD [ "/bin/sh", "-c", "python3 create_dataset.py && python3 unsloth_trainer.py && python3 merge_with_full_model.py && python3 convert_to_gguf.py --input output/merged_model --output output/Qwen2.5-7B-Instruct-fine-tuned-pharmacovigilance.gguf --quantization f32" ]
# CMD [ "/bin/sh", "-c", "python3 merge_with_full_model.py && python3 convert_to_gguf.py --input output/merged_model --output output/Qwen2.5-7B-Instruct-fine-tuned-pharmacovigilance.gguf --quantization f16" ]

# Full model
# CMD [ "/bin/sh", "-c", "python3 create_dataset.py && python3 unsloth_trainer.py && python3 convert_to_gguf.py --input output/full_model --output output/Qwen2.5-7B-Instruct-fine-tuned-pharmacovigilance.gguf --quantization f32" ]
