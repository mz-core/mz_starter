#!/usr/bin/env python3
"""Atualiza o catálogo público; execute com gh autenticado e revise o diff."""
import concurrent.futures
import json
import os
from pathlib import Path
import re
import subprocess
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
API = "https://api.github.com"
token = os.environ.get("GH_TOKEN") or os.environ.get("GITHUB_TOKEN")
if not token:
    token = subprocess.check_output(["gh", "auth", "token"], text=True).strip()

def get(path):
    request = urllib.request.Request(API + path, headers={"Authorization": "Bearer " + token, "Accept": "application/vnd.github+json", "User-Agent": "mz-starter-catalog"})
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            return json.load(response)
    except urllib.error.HTTPError as error:
        if error.code == 404:
            return None
        raise

catalog = json.loads((ROOT / "catalog/resources.json").read_text(encoding="utf-8"))
ids = [r["id"] for r in catalog["resources"] if r["type"] == "fivem_resource" and r["source"] != "protected"]
version = r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)"

def resolve(resource):
    repository = get("/repos/mz-core/" + resource)
    if not repository or repository.get("private"):
        return None
    release = get("/repos/mz-core/" + resource + "/releases/latest")
    if not release:
        return None
    tag = release["tag_name"]
    if release["draft"] or release["prerelease"] or not re.fullmatch(r"(?:" + re.escape(resource) + r"-)?v" + version, tag):
        raise ValueError(resource + ": tag da release inválida")
    assets = []
    for asset in release["assets"]:
        match = re.fullmatch(re.escape(resource) + r"-v(" + version + r")\.zip", asset["name"])
        if match and asset.get("state") == "uploaded" and re.fullmatch(r"sha256:[a-f0-9]{64}", asset.get("digest") or ""):
            assets.append({"id": resource, "version": match.group(1), "tag": tag, "asset": asset["name"], "sha256": asset["digest"]})
    if len(assets) != 1:
        raise ValueError(resource + ": release sem um único ZIP com SHA-256")
    return assets[0]

with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
    releases = [item for item in pool.map(resolve, ids) if item]
target = ROOT / "catalog/releases.json"
target.write_text(json.dumps({"schema_version": 1, "resources": releases}, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print(f"{len(releases)} releases públicas verificadas em {target}")
