from typing import Union, List, Optional
from fastapi import FastAPI
from pydantic import BaseModel
from transformers import AutoTokenizer, AutoModel
import torch
import torch.nn.functional as F
import uvicorn
import os

app = FastAPI()

MODEL_NAME = os.getenv("MODEL_NAME", "./models/gte-small-zh")
HOST = os.getenv("HOST", "0.0.0.0")
PORT = int(os.getenv("PORT", 8000))

device = "cuda" if torch.cuda.is_available() else "cpu"
print(f"[INFO] Using device: {device}")

print(f"[INFO] Loading model: {MODEL_NAME}")
tokenizer = AutoTokenizer.from_pretrained(MODEL_NAME)
model = AutoModel.from_pretrained(MODEL_NAME).to(device)
model.eval()
print("[INFO] Model loaded successfully")


# ✅ 完全兼容 OpenAI
class EmbeddingRequest(BaseModel):
    input: Union[str, List[str]]
    model: Optional[str] = None


@app.get("/health")
def health():
    return {"status": "ok"}


@app.post("/v1/embeddings")
def embeddings(req: EmbeddingRequest):
    # 👉 统一成 list
    if isinstance(req.input, str):
        texts = [req.input]
    else:
        texts = req.input

    batch_dict = tokenizer(
        texts,
        max_length=512,
        padding=True,
        truncation=True,
        return_tensors="pt"
    )

    batch_dict = {k: v.to(device) for k, v in batch_dict.items()}

    with torch.no_grad():
        outputs = model(**batch_dict)
        embeddings = outputs.last_hidden_state[:, 0]
        embeddings = F.normalize(embeddings, p=2, dim=1)

    # ✅ OpenAI 返回格式
    data = []
    for i, emb in enumerate(embeddings):
        data.append({
            "object": "embedding",
            "embedding": emb.cpu().tolist(),
            "index": i
        })

    return {
        "object": "list",
        "data": data,
        "model": req.model or "gte-small-zh"
    }


if __name__ == "__main__":
    print(f"[INFO] Starting server at http://{HOST}:{PORT}")
    uvicorn.run(app, host=HOST, port=PORT)