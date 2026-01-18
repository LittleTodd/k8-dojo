# 🥋 K8s-Dojo: The Kubernetes Troubleshooting Gym

> **Master Kubernetes Troubleshooting through 30+ Interactive Detective Scenarios.**

**K8s-Dojo** is a terminal-based interactive training tool designed to turn you into a Kubernetes troubleshooting expert. It simulates real-world cluster failures in a safe, local environment (using [Kind](https://kind.sigs.k8s.io/)) and challenges you to fix them.

![K8s Dojo TUI Placeholder](https://via.placeholder.com/800x400?text=K8s+Dojo+TUI+Demo)

---

## 🚀 Features

*   **31+ Real-World Scenarios**: Curated from production outages and expert interviews.
*   **Interactive TUI**: A beautiful Terminal User Interface (Bubbletea) to navigate and manage your training.
*   **Real-Time Validation**: Instant feedback loop. Fix the issue, press `c` to check, and get immediate results.
*   **Safe Playground**: Uses [Kind](https://kind.sigs.k8s.io) to spin up disposable local clusters. Break things without fear.
*   **Categorized Modules**: Targeted training in Networking, Security, Lifecycle, Storage, and Ops.
*   **Hints System**: Stuck? Toggle hints to get nudged in the right direction.

---

## 🛠️ Prerequisites

Before entering the dojo, ensure you have the following installed:

*   **Go** (1.23+): To compile the tool.
*   **Docker Reference**: Kind needs Docker to run nodes.
*   **[Kind](https://kind.sigs.k8s.io/docs/user/quick-start/)**: `brew install kind`
*   **[Kubectl](https://kubernetes.io/docs/tasks/tools/)**: `brew install kubectl`

---

## 📥 Installation

Clone the repository and build the binary:

```bash
git clone https://github.com/your-username/k8s-dojo.git
cd k8s-dojo
go build -o k8s-dojo ./cmd/k8s-dojo
```

---

## 🎮 How to Play

1.  **Start the Dojo**:
    ```bash
    ./k8s-dojo
    ```

2.  **Select Kubernetes Version**: Choose between the latest implementation or N-1 versions.
    *   *The tool will verify your local Kind cluster or create a new one automatically.*

3.  **Choose a Module**: Pick a domain to train in:
    *   🌐 **Networking**: Services, Ingress, DNS, NetworkPolicies.
    *   🔄 **Lifecycle**: Probes, InitContainers, CrashLoops.
    *   🔒 **Security**: RBAC, Contexts, ServiceAccounts.
    *   💾 **Storage**: PVCs, StorageClasses, Mounts.
    *   ⚙️ **Ops & specific**: Quotas, Limits, Kernel tweaks.

4.  **Solve the Scenario**:
    *   The tool will inject a fault into the cluster.
    *   **Open a new terminal window**.
    *   Use `kubectl` to investigate:
        ```bash
        kubectl get pods -A
        kubectl describe pod -n <namespace-name>
        kubectl logs ...
        ```
    *   Fix the issue (edit yaml, scale up, delete bad resources, etc.).

5.  **Verify**:
    *   Back in the TUI, press `c` to check your solution.
    *   If solved, celebrate! 🎉 Then press `Enter` to return to the menu.

---

## 🧩 Scenario Arsenal (31 Levels)

### 🌐 Networking Module
*   **Service Discovery**: Fix Service selectors (`net-service-selector`).
*   **Headless gRPC**: Client-side load balancing (`net-grpc-balance`).
*   **Source IP**: Preserving client IP (`net-source-ip`).
*   **DNS Latency**: Tuning `ndots` (`net-dns-ndots`).
*   **NetworkPolicy**: Blocking/Allowing DNS (`netpol-dns-block`).
*   **Ingress**: 404 Paths and TLS Errors (`ingress-path-error`, `ingress-tls-mismatch`).
*   **Service Ports**: TargetPort mismatches (`net-target-port-mismatch`).

### 🔄 Lifecycle & Scheduling Module
*   **CrashLoops**: Missing ConfigMaps, InitContainer failures (`crashloop-missing-config`, `init-container-crash`).
*   **Probes**: Liveness & Readiness misconfiguration (`probe-liveness-fail`, `probe-readiness-timeout`).
*   **Scheduling**: Node Affinity for GPU, Taints & Tolerations (`sched-node-affinity`, `sched-taint-toleration`).
*   **Termination**: Graceful shutdowns, Stuck Finalizers (`life-graceful-shutdown`, `pod-finalizer-stuck`).

### 🔒 Security Module
*   **RBAC**: Forbidden actions (`sec-rbac-forbidden`).
*   **Privileged Containers**: Policy violations (`sec-privileged-policy`).
*   **Supply Chain**: Mutable tags vs Digests (`sec-image-digest`).
*   **Permissions**: FSGroup volumes (`sec-fsgroup-denied`).
*   **ServiceAccount**: Token mounting (`sec-sa-nomount`).

### 💾 Storage Module
*   **PVCs**: Pending claims, StorageClass issues (`storage-pvc-pending`).
*   **Affinity**: Zonal conflicts (`storage-zonal-affinity`).
*   **Mounts**: SubPath overwrites (`storage-subpath-overwrite`).

### ⚙️ Ops & Resources Module
*   **OOM Kills**: QoS Classes (`kernel-oom-disable`).
*   **GitOps**: Config checksums (`ops-config-checksum`).
*   **Quotas**: Namespace limits (`resource-quota-exceeded`).
*   **LimitRanges**: Default constraint blocks (`resource-limit-range`).

---

## 🏗️ Architecture

*   **Language**: Go (Golang)
*   **UI Framework**: [Bubbletea](https://github.com/charmbracelet/bubbletea) (ELM architecture for TUI).
*   **Cluster Management**: Kind (Kubernetes inside Docker) SDK.
*   **K8s Interaction**: client-go.

## 🤝 Contributing

Scenario ideas are welcome! Please check `pkg/scenario/` for examples of how to implement the `Scenario` interface.

1.  Fork it
2.  Create your feature branch (`git checkout -b feature/amazing-scenario`)
3.  Commit your changes (`git commit -m 'Add Amazing Scenario'`)
4.  Push to the branch (`git push origin feature/amazing-scenario`)
5.  Create new Pull Request

---

*Happy Debugging!* 🥋
