# Use Python 3.10 as base image
FROM python:3.10-slim

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    python3-dev \
    && rm -rf /var/lib/apt/lists/*

# Install Python packages
RUN pip install --no-cache-dir networkx matplotlib pymongo dnspython python-dotenv scipy

# Copy your script
COPY create_graph_medications_adrs_only.py .env /app/

# Create a directory for output files
RUN mkdir -p /app/output

# Command to run the script
CMD ["python3", "create_graph_medications_adrs_only.py"]
