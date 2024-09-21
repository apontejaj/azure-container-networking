#!/bin/bash

set -ex
source ./.pipelines/multitenancy/scripts/utils.sh

function main() {
    # parse the arguments
    while [[ $# -gt 0 ]]; do
        key="$1"
        case "$key" in
        --test-dir)
            shift
            test_dir="$1"
            ;;
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
    run_tests "$test_dir" "$mt_test_cluster" "$scenario"
}

function run_tests() {
    local test_dir
    test_dir="${1}"
    local mt_test_cluster
    mt_test_cluster="${2}"
    local scenario
    scenario="${3}"

    STEP="runAKSe2e"
    cd $test_dir

    #get configvars
    export_envVars $scenario

    if [[ "$ENABLED" == "false" ]]; then
        echo "scenario: $scenario skipped"
        return 0
    fi
    
    # Get identity client ID
    export USER_ASSIGNED_CLIENT_ID=$(az identity show --resource-group "$RG" --name "$USER_ASSIGNED_IDENTITY_NAME" --query 'clientId' -o tsv)

    chmod +x ./akse2e.sh
    ./akse2e.sh $mt_test_cluster && passed="true" || passed="false"

    if [[ "$passed" == "true" ]]; then
        echo "Tests passed"
    else
        echo "Tests failed"
        return 1
    fi
}

main $@
