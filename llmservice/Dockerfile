FROM python:3.12-slim-bookworm

WORKDIR /app

ARG UV_INDEX_URL=https://pypi.org/simple

ENV PATH="/root/.local/bin:/app/.venv/bin:$PATH" \
    UV_INDEX_URL=${UV_INDEX_URL} \
    UV_LINK_MODE=copy

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        build-essential \
        curl \
        libgomp1 \
        libstdc++6 \
        libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

RUN curl -Ls https://astral.sh/uv/install.sh | sh

COPY pyproject.toml uv.lock README.md ./
RUN uv sync --extra dev --no-install-project

COPY main.py ./
COPY models/ ./models/

RUN pyinstaller \
    --noconfirm \
    --clean \
    --onefile \
    --name app \
    main.py \
    && mkdir -p /runtime \
    && mv /app/dist/app /runtime/app \
    && mv /app/models /runtime/models \
    && rm -rf \
        /app \
        /root/.cache \
        /root/.local/bin/uv \
        /root/.local/bin/uvx \
    && apt-get update \
    && apt-get purge -y --auto-remove \
        build-essential \
        curl \
    && rm -rf /var/lib/apt/lists/*

EXPOSE 8000

WORKDIR /runtime

CMD ["./app"]
