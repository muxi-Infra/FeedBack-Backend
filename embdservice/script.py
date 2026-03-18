from huggingface_hub import snapshot_download
# 用来下载模型的代码,上传之前需要做一个简单的拉取
snapshot_download(
    repo_id="thenlper/gte-small-zh",
    local_dir="./models/gte-small-zh",
    local_dir_use_symlinks=False
)