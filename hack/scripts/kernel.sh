CLUSTER_TYPE=overlay-byocni-up
echo $CLUSTER_TYPE
REGION=eastus
echo $REGION

echo "Subscription is - $SUB"
CLUSTER_NAME="jpayne-kernel-$(date "+%d%H%M")"
echo $CLUSTER_NAME
VMSIZE=Standard_B2ms
echo $VMSIZE
AUTOUPGRADE=none
echo $AUTOUPGRADE

make -C ../aks ${CLUSTER_TYPE} \
AZCLI=az REGION=${REGION} SUB=${SUB} \
CLUSTER=${CLUSTER_NAME} \
VM_SIZE=${VMSIZE} \
AUTOUPGRADE=none

echo "-- Install CNI/CNS --"
kubectl get pods -Aowide
kubectl apply -f https://raw.githubusercontent.com/Azure/azure-container-networking/v1.5.3/hack/manifests/cni-installer-v1.yaml
kubectl rollout status ds -n kube-system azure-cni



echo "-- Start privileged daemonset --"
kubectl get pods -Aowide
kubectl apply -f ../../test/integration/manifests/load/privileged-daemonset.yaml
sleep 10s
kubectl rollout status ds -n kube-system privileged-daemonset

echo "-- Update kernel through daemonset --"
kubectl get pods -n kube-system -l os=linux,app=privileged-daemonset -owide
privList=`kubectl get pods -n kube-system -l os=linux,app=privileged-daemonset -owide --no-headers | awk '{print $1}'`
for pod in $privList; do
    echo "-- Update Ubuntu Packages --"
    # Not needed, but ensures that the correct packages exist to perform upgrade
    kubectl exec -i -n kube-system $pod -- bash -c "apt update && apt-get install software-properties-common -y"

    echo "-- Add proposed repository --"
    kubectl exec -i -n kube-system $pod -- bash -c "add-apt-repository ppa:canonical-kernel-team/proposed -y"
    kubectl exec -i -n kube-system $pod -- bash -c "add-apt-repository ppa:canonical-kernel-team/proposed2 -y"

    echo "-- Check apt-cache --"
    kubectl exec -i -n kube-system $pod -- bash -c "apt-cache madison linux-azure-edge"

    echo "-- Check current Ubuntu kernel --"
    kubectl exec -i -n kube-system $pod -- bash -c "uname -r"
    kubectl get node -owide

    echo "-- Install Proposed Kernel --"
    kubectl exec -i -n kube-system $pod -- bash -c "apt install -y linux-azure-edge"
done

# Lines below only needed if ds cannot use nsenter, need updated ds permissions - SYS_ADMIN
# echo "-- Restart Nodes to finalize update --"
# val=`az vmss list -g MC_${CLUSTER_NAME}_${CLUSTER_NAME}_${REGION} --query "[].name" -o tsv`
# make -C ./hack/aks restart-vmss AZCLI=az CLUSTER=${CLUSTER_NAME} REGION=${REGION} VMSS_NAME=$val

privArray=(`kubectl get pods -n kube-system -l os=linux,app=privileged-daemonset -owide --no-headers | awk '{print $1}'`)
nodeArray=(`kubectl get pods -n kube-system -l os=linux,app=privileged-daemonset -owide --no-headers | awk '{print $7}'`)
kubectl get pods -n kube-system -l os=linux,app=privileged-daemonset -owide

i=0
for _ in ${privArray[@]}; do
    echo "-- Restarting Node ${nodeArray[i]} through ${privArray[i]} --"
    kubectl exec -i -n kube-system ${privArray[i]} -- bash -c "reboot"
    echo "-- Waiting for condition NotReady --"
    kubectl wait --for=condition=Ready=false -n kube-system pod/${privArray[i]} --timeout=90s
    echo "-- Waiting for condition Ready --"
    kubectl wait --for=condition=Ready -n kube-system pod/${privArray[i]} --timeout=90s
    ((i = i + 1))
    echo "Wait 10s for pods to settle"
    sleep 10s
done

# Add in regex check for expected kernel
kubectl rollout status ds -n kube-system privileged-daemonset
for pod in $privList; do
    echo "-- Check current Ubuntu kernel --"
    kubectl exec -i -n kube-system $pod -- bash -c "uname -r"
done
kubectl get node -owide

echo "To delete all resources use | az group delete -n ${CLUSTER_NAME} --no-wait -y"
