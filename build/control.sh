#!/bin/bash
set -e
ROOT=$(
    cd "$(dirname "$0")"
    pwd
)

###### OPERATION #######
function deploy() {
    local module=$1
    if [[ -z ${module} ]]; then
        echo "no module provided"
        exit 1
    else
        echo ">>> deploy module: ${module}"
    fi
    local tag=$2
    if [[ -z ${tag} ]]; then
        echo "no tag provided"
        exit 2
    fi
    cd ${ROOT}
    export IMAGE_TAG=${tag}
    export SELECTOR="node-role.kubernetes.io/master: master"
    envsubst '${IMAGE_TAG} ${SELECTOR}' <${ROOT}/${module}.yaml | kubectl apply -f -
}

function deployment() {
    local tag=$1
    deploy observer ${tag}
    deploy operator ${tag}
}

function get() {
    kubectl -n kml get pods -l app=ares
}

function install() {
    local files=("namespace.yaml"
        "service_account.yaml"
        "role.yaml"
        "role_binding.yaml"
        "ares.io_aresjobs.yaml"
    )
    for f in "${files[@]}"; do
        kubectl apply -f ${ROOT}/crd/${f}
    done
}

ACTION="get"
if [[ $# -ge 1 ]]; then
    ACTION=$1
fi
case ${ACTION} in
deploy)
    shift
    deployment "$@"
    ;;
install)
    shift
    install "$@"
    ;;
get)
    get
    ;;
*)
    echo "unknown action: ${ACTION}"
    echo -e "Usage:\n\t$0 get\n\t$0 install\n\t$0 deploy <image-tag>"
    exit 255
    ;;
esac
