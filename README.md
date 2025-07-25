## Getting Started

### Prerequisites
- Go (version 1.22 or later)
- protoc (Protocol Buffers Compiler)
- Go gRPC plugins (`protoc-gen-go`, `protoc-gen-go-grpc`)

### Local Development

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/sentiric/sentiric-user-service.git
    cd sentiric-user-service
    ```

2.  **Generate gRPC Code:** This project depends on `.proto` files from the `sentiric-core-interfaces` repository. You need to generate the Go code from these contracts.
    
    **Option A (Recommended): Using the central Makefile**
    - Navigate to the `sentiric-core-interfaces` repository.
    - Run the make command:
      ```bash
      make gen-go
      ```
    - This will generate the necessary files in a `gen/` directory at the root of your workspace. You may need to copy the relevant `gen/user/v1` folder into this project.
    
    **Option B (Manual Generation):**
    - Ensure `sentiric-core-interfaces` is cloned next to this repository.
    - Run the `protoc` command directly:
      ```bash
      mkdir -p gen/user/v1
      protoc --proto_path=../sentiric-core-interfaces/proto \
             --go_out=./gen --go_opt=paths=source_relative \
             --go-grpc_out=./gen --go-grpc_opt=paths=source_relative \
             ../sentiric-core-interfaces/proto/sentiric/user/v1/user.proto
      ```

3.  **Run the service:**
    ```bash
    go run main.go
    ```