# Persistent Volume & NFS Setup

This module sets up the shared persistent volume used by various services.

## Production Setup (NFS Server)
On your NFS server (e.g. at IP `74.220.21.187`):
```bash
sudo apt-get update
sudo apt-get install -y nfs-kernel-server
sudo mkdir -p /mnt/nfs_share/
sudo chown nobody:nogroup /mnt/nfs_share/
sudo chmod 777 /mnt/nfs_share/
echo "/mnt/nfs_share *(rw,sync,no_subtree_check,no_root_squash)" | sudo tee -a /etc/exports
sudo exportfs -a
sudo systemctl restart nfs-kernel-server
```

## Local Cluster Deployment (Minikube / Docker Desktop)
Uncomment the `hostPath` block in `pv.yaml` and comment out the `nfs` block before deploying.
