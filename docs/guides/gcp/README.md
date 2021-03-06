# Example: Kubo on GCP

## Prerequisites

Configure a GCP project and deploy a BOSH bastion by following the "Configure your Google Cloud Platform environment" and "Deploy supporting infrastructure" steps in
  [these instructions](https://github.com/cloudfoundry-incubator/bosh-google-cpi-release/blob/c2cdba4f2ac8944ce7eb9749f053d45588932e3b/docs/bosh/README.md).

## Prepare GCP Infrastructure

The remaining steps should all be done in succession from a single session to retain required environment variables.

1. ssh into the bosh-bastion created in the prerequisites
   ```bash
   gcloud compute ssh bosh-bastion
   ```

1. Start from the home directory of the bosh-bastion:
   ```bash
   cd
   ```

1. Install deployment dependencies:
   ```bash
   sudo curl https://s3.amazonaws.com/bosh-cli-artifacts/bosh-cli-2.0.18-linux-amd64 -o /usr/bin/bosh-cli
   sudo chmod a+x /usr/bin/bosh-cli
   curl -L https://github.com/cloudfoundry-incubator/credhub-cli/releases/download/0.4.0/credhub-linux-0.4.0.tgz | tar zxv
   chmod a+x credhub
   sudo mv credhub /usr/bin
   sudo curl -L https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl -o /usr/bin/kubectl
   sudo chmod a+x /usr/bin/kubectl
   ```


1. Clone the [kubo-deployment](https://github.com/pivotal-cf-experimental/kubo-deployment) repo
   ```bash
   git clone https://github.com/pivotal-cf-experimental/kubo-deployment.git
   ```

1. `cd` into the guide directory

   ```bash
   cd ~/kubo-deployment/docs/guides/gcp
   ```

1. Export these values. If you haven't tweaked any settings then use these defaults:

   ```bash
   export project_id=$(gcloud config get-value project)
   export kubo_region=us-west1
   export kubo_zone=us-west1-a
   export kubo_env=kube
   export state_dir=~/kubo-env/${kubo_env}
   export kubo_terraform_state=${state_dir}/terraform.tfstate
   export network=<network created using terraform script above. By default - bosh>
   ``` 

1. Create a folder to store the environment configuration
   ```bash
   mkdir -p ${state_dir} 
   ```

1. View the Terraform execution plan to see the resources that will be created:
   ```bash
   terraform plan \
      -var network=${network} \
      -var projectid=${project_id} \
      -var kubo_region=${kubo_region} \
      -state=${kubo_terraform_state}
   ```  

1. Create the resources
   ```bash
   terraform apply \
      -var network=${network} \
      -var projectid=${project_id} \
      -var kubo_region=${kubo_region} \
      -state=${kubo_terraform_state}
   ```

Additionally, the terraform script accepts the following variables:
  
  - `ip_cidr_range`: the CIDR range for the kubo subnetwork. The default value is `10.0.1.0/24`
  - `prefix`: A prefix to use for all the GCP resource names. Defaults to an empty string.
  
## Configure Kubo

1. Retrieve the outputs of your Terraform run to be used in your Kubo deployment

   ```bash
   export kubo_subnet=$(terraform output -state=${kubo_terraform_state} kubo_subnet)
   export kubo_master_target_pool=$(terraform output -state=${kubo_terraform_state} kubo_master_target_pool)
   export kubo_worker_target_pool=$(terraform output -state=${kubo_terraform_state} kubo_worker_target_pool)
   export kubernetes_api_ip="$(terraform output -state=${kubo_terraform_state} master_lb_ip_address)"
   ```

1. Populate the director configurations
   ```bash
   erb director.yml.erb > ${state_dir}/director.yml
   ```

1. Generate a service account key for the bosh-user
   ```bash
   export service_account=bosh-user
   export service_account_creds=${state_dir}/service_account.json
   export service_account_email=${service_account}@${project_id}.iam.gserviceaccount.com
   gcloud iam service-accounts keys create ${service_account_creds} --iam-account ${service_account_email}
   ```

## Deploy Kubo

1. Return to the root of the kubo-deployment repo

   ```bash
   cd ~/kubo-deployment
   ```

1. Deploy a BOSH director for Kubo
   ```bash
   bin/deploy_bosh ${state_dir} ${service_account_creds} 
   ```

1. Deploy Kubo
   ```bash
   bin/deploy_k8s ${state_dir} kube public
   ```

1. Setup kubectl and access your new Kubernetes cluster
   ```bash
   bin/set_kubeconfig ${state_dir} kube
   kubectl get pods --namespace=kube-system
   ```

## Accessing Kubernetes services

To expose services running in your Kubernetes cluster, use the service type `NodePort` when deploying your service to Kubernetes. We do not currently support the type `LoadBalancer`, but we plan to soon with Github issue [#47](https://github.com/pivotal-cf-experimental/kubo-release/issues/47) in the [kubo-release](https://github.com/pivotal-cf-experimental/kubo-release) repository. Until this issue is resolved, an additional load balancer is provisioned using Terraform during the setup in our guide. If your service is exposed with a NodePort, you can access the service using the external IP address of the kubo-workers load balancer and the node port of your service.

### Example: Accessing the Kubernetes dashboard on GCP
   
1. Find the IP address of your worker load balancer

   ```
   gcloud compute addresses list | grep kubo-workers

     kubo-workers    us-west1  XX.XXX.X.XXX     IN_USE
   ```

1. Find the Node Port of the kubernetes-dashboard service

   ```
   kubectl describe service kubernetes-dashboard --namespace kube-system  | grep NodePort

     Type:                   NodePort
     NodePort:               <unset> 31000/TCP
   ```

1. Access your service from your browser at `<IP address of load balancer>:<NodePort>`
