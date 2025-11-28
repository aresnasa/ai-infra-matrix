import os
import re

substitutions = {
    "GOLANG_ALPINE_VERSION": "GOLANG_ALPINE_VERSION",
    "GOLANG_VERSION": "GOLANG_VERSION",
    "NODE_ALPINE_VERSION": "NODE_ALPINE_VERSION",
    "NGINX_VERSION": "NGINX_VERSION",
    "NGINX_ALPINE_VERSION": "NGINX_ALPINE_VERSION",
    "UBUNTU_VERSION": "UBUNTU_VERSION",
    "SLURM_VERSION": "SLURM_VERSION",
    "ALPINE_MIRROR": "ALPINE_MIRROR",
    "GO_PROXY": "GO_PROXY",
    "NPM_REGISTRY": "NPM_REGISTRY",
    "APT_MIRROR": "APT_MIRROR",
    "YUM_MIRROR": "YUM_MIRROR",
    "PIP_VERSION": "PIP_VERSION",
    "PYPI_INDEX_URL": "PYPI_INDEX_URL",
    "HAPROXY_VERSION": "HAPROXY_VERSION",
    "SALTSTACK_VERSION": "SALTSTACK_VERSION",
    "CATEGRAF_VERSION": "CATEGRAF_VERSION",
    "SINGULARITY_VERSION": "SINGULARITY_VERSION",
    "PYTHON_ALPINE_VERSION": "PYTHON_ALPINE_VERSION",
    "GITEA_VERSION": "GITEA_VERSION",
    "JUPYTER_BASE_NOTEBOOK_VERSION": "JUPYTER_BASE_NOTEBOOK_VERSION",
    "ROCKYLINUX_VERSION": "ROCKYLINUX_VERSION",
    "ALPINE_VERSION": "ALPINE_VERSION",
}

# Special handling for FROM instructions that are hardcoded
from_overrides = {
    "ubuntu:22.04": "ubuntu:{{UBUNTU_VERSION}}",
    "jupyter/base-notebook:latest": "jupyter/base-notebook:{{JUPYTER_BASE_NOTEBOOK_VERSION}}",
}

def process_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    # Replace ARG VAR=...
    for var in substitutions:
        # Case 1: ARG VAR=value
        pattern = fr"ARG {var}=.*"
        if re.search(pattern, content):
            content = re.sub(pattern, f"ARG {var}={{{{{var}}}}}", content)
        
        # Case 2: ARG VAR (without value)
        pattern_no_val = fr"ARG {var}\s*$"
        if re.search(pattern_no_val, content, re.MULTILINE):
             content = re.sub(pattern_no_val, f"ARG {var}={{{{{var}}}}}", content, flags=re.MULTILINE)

    # Replace FROM overrides
    for old, new in from_overrides.items():
        content = content.replace(f"FROM {old}", f"FROM {new}")

    tpl_path = filepath + ".tpl"
    with open(tpl_path, 'w') as f:
        f.write(content)
    print(f"Created {tpl_path}")

# Walk src directory
for root, dirs, files in os.walk("src"):
    for file in files:
        if file == "Dockerfile":
            process_file(os.path.join(root, file))
