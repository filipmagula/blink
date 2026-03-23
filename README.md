# Blink

Blink is a lightweight, fast, and simple file hosting and sharing application. It features a Go backend for handling file uploads/downloads and a Vue-based admin dashboard for managing files and viewing server statistics.

## Features

- **Simple File Upload**: Upload files via PUT requests.
- **Admin Dashboard**: Manage uploaded files and view usage statistics.
- **Secure**: Basic authentication for admin routes and JWT-based session management.
- **Lightweight**: Built with Go and Vue, running in a minimal Alpine-based Docker container.

## How to Run

The easiest way to run Blink is using Docker Compose.

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)

### Quick Start

1.  **Clone the repository** (if you haven't already).
2.  **Start the application**:
    ```bash
    docker-compose up --build -d
    ```
3.  **Access the application**:
    - The server will be running on port **21337**.
    - **Admin Dashboard**: [http://localhost:21337/admin/](http://localhost:21337/admin/)
    - **Upload a file**:
      ```bash
      curl -T yourfile.txt http://localhost:21337/
      ```

### Default Credentials

- **Admin Password**: `supersecretadmin` (Change this in `docker-compose.yml` for production!)

### Environment Variables

You can securely manage secrets using a `.env` file. Docker Compose will automatically load variables from `.env`.

1. **Create your environment file**:
   Copy the example file to create your own configuration.
   ```bash
   cp .env.example .env
   ```
2. **Edit `.env`** with your secure credentials:
   ```env
   ADMIN_PASSWORD=supersecretadmin
   JWT_SECRET=generate-a-long-random-string-here
   ```
3. **Start the application**:
   ```bash
   docker compose up -d
   ```

> [!IMPORTANT]
> Do not commit your `.env` file to version control. It is ignored by git by default. If you don't provide these variables, the application will use its internal defaults.
