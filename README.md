# Sentiric User Service

**Description:** Manages user accounts, authentication credentials, and active SIP registrations securely and persistently for the Sentiric platform.

**Core Responsibilities:**
*   Storing and updating active SIP registrations (Contact URI, expiration time, associated IP/port).
*   Performing SIP Digest Authentication based on user credentials (HA1 hashes).
*   Providing APIs for CRUD (Create, Read, Update, Delete) operations on user accounts.
*   Finding user information by extension or username.

**Technologies:**
*   Node.js (or Go)
*   Express/Fiber (for REST API)
*   Database connection (e.g., PostgreSQL, MongoDB, Redis).

**API Interactions (As an API Provider):**
*   Exposes APIs for `sentiric-sip-server` (for authentication, registration updates, user lookups).
*   Exposes APIs for `sentiric-admin-ui` (for user CRUD operations).

**Local Development:**
1.  Clone this repository: `git clone https://github.com/sentiric/sentiric-user-service.git`
2.  Navigate into the directory: `cd sentiric-user-service`
3.  Install dependencies: `npm install` (Node.js) or `go mod tidy` (Go).
4.  Create a `.env` file from `.env.example` to configure database connections and authentication realm.
5.  Start the service: `npm start` (Node.js) or `go run main.go` (Go).

**Configuration:**
Refer to `config/` directory and `.env.example` for service-specific configurations, including database connection details and authentication realm.

**Deployment:**
Designed for containerized deployment (e.g., Docker, Kubernetes). Refer to `sentiric-infrastructure`.

**Contributing:**
We welcome contributions! Please refer to the [Sentiric Governance](https://github.com/sentiric/sentiric-governance) repository for coding standards and contribution guidelines.

**License:**
This project is licensed under the [License](LICENSE).
