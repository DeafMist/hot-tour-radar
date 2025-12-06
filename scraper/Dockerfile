FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy source code
COPY ./src/parsers/*.py ./src/parsers/
COPY ./src/producers/*.py ./src/producers/
COPY ./src/*.py ./src/

# Entrance point
CMD ["python", "-m", "src.main"]
