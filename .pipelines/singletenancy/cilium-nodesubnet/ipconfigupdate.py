import json
import subprocess
import argparse

parser = argparse.ArgumentParser(description="Update IP configuration for VMSS.")
parser.add_argument("--resource-group", required=True, help="The resource group of the VMSS.")
parser.add_argument("--secondary-config-count", type=int, required=True, help="The count of secondary IP configurations.")
args = parser.parse_args()
resource_group = args.resource_group
secondary_config_count = args.secondary_config_count

command = f"az vmss list -g {resource_group} --query '[0].name' -o tsv"
result = subprocess.run(command, shell=True, capture_output=True, text=True)

if result.returncode == 0:
    vmss_name = result.stdout.strip()
else:
    raise Exception(f"Command failed with error: {result.stderr}")

command = f"az vmss show -g {resource_group} -n {vmss_name}"
result = subprocess.run(command, shell=True, capture_output=True, text=True)

if result.returncode == 0:
    vmss_info = json.loads(result.stdout)
else:
    raise Exception(f"Command failed with error: {result.stderr}")

used_ip_config_names = []    
secondary_configs = []

if "virtualMachineProfile" in vmss_info and "networkProfile" in vmss_info["virtualMachineProfile"]:
    network_profile = vmss_info["virtualMachineProfile"]["networkProfile"]
    if "networkInterfaceConfigurations" in network_profile:
        for nic_config in network_profile["networkInterfaceConfigurations"]:
            primary_ip_config = None

            if "ipConfigurations" in nic_config:
                for ip_config in nic_config["ipConfigurations"]:
                    if "name" in ip_config:
                        used_ip_config_names.append(ip_config["name"])

                    if "primary" in ip_config and ip_config["primary"]:
                        primary_ip_config = ip_config

                if primary_ip_config is not None:        
                    for i in range(2, secondary_config_count + 2):
                        ip_config = primary_ip_config.copy()
                        if f"ipconfig{i}" not in used_ip_config_names:
                            ip_config["name"] = f"ipconfig{i}"
                            ip_config["primary"] = False
                            used_ip_config_names.append(ip_config["name"])
                            secondary_configs.append(ip_config)

                nic_config["ipConfigurations"].extend(secondary_configs)

                        
    command = f"az vmss update -g {resource_group} -n {vmss_name} --set virtualMachineProfile.networkProfile='{json.dumps(network_profile)}'"
    print("Command to update VMSS: ", command)
    result = subprocess.run(command, shell=True)
    if result.returncode != 0:
        raise Exception(f"Command failed with error: {result.stderr}")

    command = f"az vmss update-instances -g {resource_group} -n {vmss_name} --instance-ids '*'"
    print("Command to update VMSS instances: ", command)
    result = subprocess.run(command, shell=True)
    if result.returncode != 0:
        raise Exception(f"Command failed with error: {result.stderr}")