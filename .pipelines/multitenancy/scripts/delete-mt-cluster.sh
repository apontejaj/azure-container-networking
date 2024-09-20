#!/bin/bash

set -ex
source ./.pipelines/multitenancy/scripts/utils.sh

function main() {
    # parse the arguments
    while [[ $# -gt 0 ]]; do
        key="$1"
        case $key in
        --mt-test-cluster)
            shift
            mt_test_cluster="$1"
            ;;
        --scenario)
            shift
            scenario="$1"
            ;;
        *)
            echo 1>&2 "unknown argument: $1"
            return 1
            ;;
        esac
        shift
    done
    delete_aks_mt_cluster "$mt_test_cluster" "$scenario"
}

cluster_exists() {
    (az aks show --name $mt_test_cluster -g $RG >/dev/null) && echo "true" || echo "false"
}

function delete_aks_mt_cluster() {
    local mt_test_cluster
    mt_test_cluster="${2}"
    local scenario
    scenario="${3}"

    export_envVars "$scenario"

    if [[ "$ENABLED" == "false" ]]; then
        echo "scenario: $scenario skipped"
        return 0
    fi

    STEP="DeleteMTAKSCluster"
    # check if cluster exists, delete if so
    deleted="false"

    for attempt in $(seq 1 3); do
        echo "checking for cluster deletion, attempt: $attempt/3"
        exists="$(cluster_exists)"

        if [[ "$exists" == "true" ]]; then
            az aks delete -n "$mt_test_cluster" -g "$RG" --yes
            exists="$(cluster_exists)"
        fi
        if [[ "$exists" == "false" ]]; then
            echo "cluster is deleted"
            deleted="true"
            break
        fi
        sleep 30
    done

    #Clean up user-assigned identity
    az identity federated-credential delete --name "$FEDERATED_IDENTITY_CREDENTIAL_PREFIX-$mt_test_cluster" --identity-name "$USER_ASSIGNED_IDENTITY_NAME" --resource-group "$RG" --yes
}

main $@
