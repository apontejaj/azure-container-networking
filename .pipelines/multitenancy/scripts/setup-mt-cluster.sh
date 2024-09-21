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
    setup_aks_mt_cluster "$mt_test_cluster" "$scenario"
}

function setup_aks_mt_cluster() {
    local mt_test_cluster
    mt_test_cluster="${1}"
    local scenario
    scenario="${2}"

    #get configvars
    export_envVars "${scenario}"

    if [[ "$ENABLED" == "false" ]]; then
        echo "scenario: $scenario skipped"
        return 0
    fi

    # Create multitenant cluster
    STEP="CreateMTAKSOverlayCluster"
    az aks create --name $mt_test_cluster \
        -g $RG \
        --tags runnercluster=true stampcreatorserviceinfo=true \
        --network-plugin azure \
        --network-plugin-mode overlay \
        --location $LOCATION \
        --node-count 2 \
        --node-vm-size $VM_SIZE \
        --node-os-upgrade-channel NodeImage \
        --kubernetes-version 1.30 \
        --nodepool-name "mtapool0" \
        --nodepool-tags fastpathenabled=true aks-nic-enable-multi-tenancy=true \
        --enable-oidc-issuer \
        --enable-workload-identity \
        --generate-ssh-keys && passed="true" || passed="false"
    
    # Get the OIDC Issuer URL
    export AKS_OIDC_ISSUER="$(az aks show -n "$mt_test_cluster" -g "$RG" --query "oidcIssuerProfile.issuerUrl" -otsv)"

    # Federate the identity
    az identity federated-credential create \
        --name "$FEDERATED_IDENTITY_CREDENTIAL_PREFIX-$mt_test_cluster" \
        --identity-name "$USER_ASSIGNED_IDENTITY_NAME" \
        --resource-group "$RG" \
        --issuer "$AKS_OIDC_ISSUER" \
        --subject system:serviceaccount:mtpod-to-service-endpoint:workload-identity-sa

    if [[ "$passed" == "true" ]]; then
        echo "Tests passed"
    else
        echo "Tests failed"
        return 1
    fi
}

main $@
