from typing import Union, List
from fastapi import FastAPI
from pydantic import BaseModel
import numpy as np
import os
import uvicorn
import onnxruntime as ort
from transformers import AutoTokenizer

app = FastAPI()

# 环境变量
EMBEDDING_MODEL_PATH = os.getenv("EMB_PATH", "./models/gte-small-zh-int8")
NLI_MODEL_PATH = os.getenv("NLI_PATH", "./models/nli-mini-int8")
HOST = os.getenv("HOST", "0.0.0.0")
PORT = int(os.getenv("PORT", 8000))

# ONNX Provider
provider = "CPUExecutionProvider"

# 加载 Tokenizer

print(f"[INFO] Loading tokenizer: {EMBEDDING_MODEL_PATH}")
emb_tokenizer = AutoTokenizer.from_pretrained(EMBEDDING_MODEL_PATH)

print(f"[INFO] Loading tokenizer: {NLI_MODEL_PATH}")
nli_tokenizer = AutoTokenizer.from_pretrained(NLI_MODEL_PATH)

# 加载 ONNX 模型

print(f"[INFO] Loading ONNX Embedding model...")
emb_session = ort.InferenceSession(
    os.path.join(EMBEDDING_MODEL_PATH, "model.onnx"),
    providers=[provider]
)

print(f"[INFO] Loading ONNX NLI model...")
nli_session = ort.InferenceSession(
    os.path.join(NLI_MODEL_PATH, "model.onnx"),
    providers=[provider]
)


# 工具函数
def softmax(x):
    e_x = np.exp(x - np.max(x))
    return e_x / e_x.sum(axis=-1, keepdims=True)


def l2_normalize(x):
    norm = np.linalg.norm(x, axis=1, keepdims=True)
    return x / norm



# 数据模型
class EmbeddingRequest(BaseModel):
    input: Union[str, List[str]]


class NLICheckRequest(BaseModel):
    premise: str
    hypothesis: str



# Embedding 接口
@app.post("/v1/embeddings")
async def embeddings(req: EmbeddingRequest):
    texts = [req.input] if isinstance(req.input, str) else req.input

    inputs = emb_tokenizer(
        texts,
        padding=True,
        truncation=True,
        max_length=512,
        return_tensors="np"
    )

    # ONNX 推理
    outputs = emb_session.run(None, dict(inputs))

    # outputs[0] 通常是 last_hidden_state
    last_hidden_state = outputs[0]

    # CLS pooling
    embeddings = last_hidden_state[:, 0, :]
    embeddings = l2_normalize(embeddings)

    data = [
        {"object": "embedding", "embedding": emb.tolist(), "index": i}
        for i, emb in enumerate(embeddings)
    ]

    return {"object": "list", "data": data}


# NLI 接口
@app.post("/v1/nli/check")
async def nli_check(req: NLICheckRequest):
    inputs = nli_tokenizer(
        req.premise,
        req.hypothesis,
        return_tensors="np",
        truncation=True,
        max_length=512
    )

    outputs = nli_session.run(None, dict(inputs))

    logits = outputs[0]
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


# 启动服务
if __name__ == "__main__":
    uvicorn.run(app, host=HOST, port=PORT)