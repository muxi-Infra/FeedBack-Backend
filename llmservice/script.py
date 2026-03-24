import os
import shutil
from pathlib import Path

# 1. 镜像源配置
os.environ["HF_ENDPOINT"] = "https://hf-mirror.com"
os.environ["HF_HUB_OFFLINE"] = "0"

from optimum.onnxruntime import (
    ORTModelForFeatureExtraction,
    ORTModelForSequenceClassification,
    ORTQuantizer
)
from optimum.onnxruntime.configuration import AutoQuantizationConfig
from transformers import AutoTokenizer


def minimal_export_and_quantize(repo_id, local_path, task_type="feature-extraction"):
    # 临时存放原始 ONNX 的路径
    temp_path = f"{local_path}_TEMP_RAW"
    final_dir = Path(local_path)

    print(f"\n📦 [1/3] 导出原始模型至临时目录...")
    try:
        # 导出原始 FP32 ONNX
        if task_type == "feature-extraction":
            model = ORTModelForFeatureExtraction.from_pretrained(repo_id, export=True, token=False)
        else:
            model = ORTModelForSequenceClassification.from_pretrained(repo_id, export=True, token=False)

        model.save_pretrained(temp_path)
        tokenizer = AutoTokenizer.from_pretrained(repo_id, token=False)
        tokenizer.save_pretrained(temp_path)

        print(f"💎 [2/3] 执行 INT8 量化...")
        quantizer = ORTQuantizer.from_pretrained(temp_path)
        dqconfig = AutoQuantizationConfig.avx512_vnni(is_static=False, per_channel=False)

        # 将量化结果直接存入最终目录
        quantizer.quantize(save_dir=local_path, quantization_config=dqconfig)
        tokenizer.save_pretrained(local_path)

        print(f"🧹 [3/3] 正在进行极致瘦身（清理中间件与重命名）...")

        # 关键操作：将量化后的文件重命名为标准文件名，覆盖/替换掉可能的旧文件
        q_file = final_dir / "model_quantized.onnx"
        std_file = final_dir / "model.onnx"

        if q_file.exists():
            if std_file.exists():
                os.remove(std_file)  # 如果存在原始的大文件，删掉
            os.rename(q_file, std_file)  # 把量化版“扶正”
            print(f"✅ 已将量化模型重命名为标准 model.onnx")

        # 彻底删除临时文件夹
        if os.path.exists(temp_path):
            shutil.rmtree(temp_path)

        print(f"✨ 处理完成！最终目录 {local_path} 仅保留 INT8 核心。")

    except Exception as e:
        print(f"❌ 出错: {e}")
        if os.path.exists(temp_path):
            shutil.rmtree(temp_path)


if __name__ == "__main__":
    tasks = [
        {"repo_id": "thenlper/gte-small-zh", "path": "./models/gte-small-zh-int8", "type": "feature-extraction"},
        {"repo_id": "cross-encoder/nli-MiniLM2-L6-H768", "path": "./models/nli-mini-int8",
         "type": "sequence-classification"}
    ]

    for task in tasks:
        minimal_export_and_quantize(task["repo_id"], task["path"], task["type"])