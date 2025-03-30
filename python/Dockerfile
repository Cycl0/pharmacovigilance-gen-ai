# Base python 3.9 slim
FROM python:3.9-slim

# Diretorio base
WORKDIR /app

# Copiar arquivos necessarios
COPY get_posts.py .
COPY requirements.txt .
COPY .env .

# Instalar dependencias
RUN pip install --no-cache-dir -r requirements.txt

# Executar o script quando o container iniciar
["python", "get_posts.py"]
