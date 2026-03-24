from typing import Union, List, Optional
from fastapi import FastAPI
from pydantic import BaseModel
import numpy as np
import os
import uvicorn

# 核心：使用 optimum 提供的 ONNX 运行运行时
from optimum.onnxruntime import ORTModelForFeatureExtraction, ORTModelForSequenceClassification
from transformers import AutoTokenizer

app = FastAPI()

# 环境变量配置 (指向包含 .onnx 文件的文件夹)
EMBEDDING_MODEL_PATH = os.getenv("EMB_PATH", "./models/gte-small-zh-int8")
NLI_MODEL_PATH = os.getenv("NLI_PATH", "./models/nli-mini-int8")
HOST = os.getenv("HOST", "0.0.0.0")
PORT = int(os.getenv("PORT", 8000))

# 选择推理引擎 (CPU 或 CUDA)
# 如果没有 GPU，ONNX 会自动回退到 CPU，且依然比原生 PyTorch 快得多
provider = "CPUExecutionProvider"

# --- 1. 加载 ONNX 化的 Embedding 模型 (GTE) ---
print(f"[INFO] Loading ONNX Embedding model from: {EMBEDDING_MODEL_PATH}")
emb_tokenizer = AutoTokenizer.from_pretrained(EMBEDDING_MODEL_PATH)
emb_model = ORTModelForFeatureExtraction.from_pretrained(
    EMBEDDING_MODEL_PATH,
    provider=provider
)

# --- 2. 加载 ONNX 化的 NLI 模型 (MiniLM) ---
print(f"[INFO] Loading ONNX NLI model from: {NLI_MODEL_PATH}")
nli_tokenizer = AutoTokenizer.from_pretrained(NLI_MODEL_PATH)
nli_model = ORTModelForSequenceClassification.from_pretrained(
    NLI_MODEL_PATH,
    provider=provider
)


# --- 工具函数：Softmax 和 Normalize ---
def softmax(x):
    e_x = np.exp(x - np.max(x))
    return e_x / e_x.sum(axis=-1, keepdims=True)


def l2_normalize(x):
    norm = np.linalg.norm(x, axis=1, keepdims=True)
    return x / norm


# --- 数据模型 ---
class EmbeddingRequest(BaseModel):
    input: Union[str, List[str]]


class NLICheckRequest(BaseModel):
    premise: str
    hypothesis: str


# --- 路由实现 ---

@app.post("/v1/embeddings")
async def embeddings(req: EmbeddingRequest):
    texts = [req.input] if isinstance(req.input, str) else req.input
    inputs = emb_tokenizer(texts, padding=True, truncation=True, max_length=512, return_tensors="np")

    # ONNX 推理
    outputs = emb_model(**inputs)
    # GTE 通常取第一个 token (CLS) 的输出
    last_hidden_state = outputs.last_hidden_state
    embeddings = last_hidden_state[:, 0, :]
    embeddings = l2_normalize(embeddings)

    data = [{"object": "embedding", "embedding": emb.tolist(), "index": i}
            for i, emb in enumerate(embeddings)]
    return {"object": "list", "data": data}


@app.post("/v1/nli/check")
async def nli_check(req: NLICheckRequest):
    # ONNX Runtime 处理长文本非常高效
    inputs = nli_tokenizer(
        req.premise,
        req.hypothesis,
        return_tensors="np",  # 注意这里返回 numpy 格式
        truncation=True,
        max_length=512
    )

    outputs = nli_model(**inputs)
    logits = outputs.logits
    probs = softmax(logits)[0]

    res = {
        "is_valid": bool(probs[1] > 0.5),
        "scores": {
            "contradiction": float(probs[0]),
            "entailment": float(probs[1]),
            "neutral": float(probs[2])
        }
    }

    if probs[0] > 0.4:
        res["is_valid"] = False
        res["status"] = "violation"
    else:
        res["status"] = "pass"

    return res


if __name__ == "__main__":
    uvicorn.run(app, host=HOST, port=PORT)