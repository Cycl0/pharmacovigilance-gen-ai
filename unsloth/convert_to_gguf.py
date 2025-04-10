#!/usr/bin/env python3
import os
import subprocess
import sys
import argparse

def convert_to_gguf(input_model_path, output_gguf_path, quantization="f16"):
    """
    Convert a model to GGUF format using llama.cpp's convert-hf-to-gguf.py.
    
    Args:
        input_model_path: Path to the input model directory
        output_gguf_path: Path where to save the GGUF model
        quantization: Format to use (default: f16)
    """
    print("Attempting to convert model to GGUF format...")

    # Verify input model exists
    if not os.path.exists(input_model_path):
        raise FileNotFoundError(f"Input model path does not exist: {input_model_path}")

    # Store original directory
    original_dir = os.getcwd()

    try:
        # Clone llama.cpp if not exists
        if not os.path.exists("llama.cpp"):
            print("Cloning llama.cpp repository...")
            subprocess.run(["git", "clone", "https://github.com/ggerganov/llama.cpp.git"], check=True)
        
        # Change to llama.cpp directory
        os.chdir("llama.cpp")
        
        # Make the convert-hf-to-gguf.py script executable
        converter_path = "convert_hf_to_gguf.py"
        if not os.access(converter_path, os.X_OK):
            os.chmod(converter_path, 0o755)
        
        # Convert the model using convert-hf-to-gguf.py
        print(f"Converting model with format: {quantization}")
        full_input_path = os.path.abspath(os.path.join(original_dir, input_model_path))
        full_output_path = os.path.abspath(os.path.join(original_dir, output_gguf_path))
        
        try:
            subprocess.run([
                "python3",
                converter_path,
                full_input_path,
                "--outfile", full_output_path,
                "--outtype", quantization
            ], check=True)
            print(f"Model successfully converted to GGUF format and saved as {full_output_path}")
        except subprocess.CalledProcessError as e:
            print(f"Error during conversion: {str(e)}")
            print("Command output:")
            print(e.output if e.output else "No output available")
            raise
    
    finally:
        # Always return to original directory
        os.chdir(original_dir)

def main():
    parser = argparse.ArgumentParser(description='Convert a model to GGUF format')
    parser.add_argument('--input', type=str, default=os.path.join("output", "final_model"),
                        help='Path to the input model directory (default: output/final_model)')
    parser.add_argument('--output', type=str, 
                        default=os.path.join("output", "final_model.gguf"),
                        help='Path where to save the GGUF model')
    parser.add_argument('--quantization', type=str, default="f16",
                        help='Format to use (default: f16)')
    
    args = parser.parse_args()
    
    try:
        # Convert the model
        convert_to_gguf(args.input, args.output, args.quantization)
    except Exception as e:
        print(f"Error: {str(e)}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main() 