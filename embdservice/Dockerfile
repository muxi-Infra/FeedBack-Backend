# syntax=docker/dockerfile:1

FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    HOST=0.0.0.0 \
    PORT=8000 \
    MODEL_NAME=/app/models/gte-small-zh

WORKDIR /app

# system deps (git 可选；很多 huggingface 资源不需要 git，但保留更稳)
RUN apt-get update \
    && apt-get install -y --no-install-recommends git \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt ./

# 安装 python deps（torch 这里尝试默认安装 CPU 版；如需 CUDA，请改成你自己的基础镜像/安装指令）
RUN pip install --no-cache-dir -U pip \
    && pip install --no-cache-dir -r requirements.txt \
    && pip install --no-cache-dir torch

COPY . .

EXPOSE 8000

CMD ["python", "main.py"]

