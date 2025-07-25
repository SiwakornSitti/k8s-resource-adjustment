# K8S Resource Adjustment

A Go-based automation tool that programmatically updates Kubernetes resource settings (CPU/memory requests and limits) in YAML files within one or more Git repositories. It clones the repositories, modifies the files in-memory, and pushes the changes back to the remote.

## Features

- **Automated Resource Updates**: Modifies CPU and memory requests and limits for various Kubernetes kinds, including Deployments, DaemonSets, StatefulSets, Pods, and Jobs.
- **Multi-Repository Support**: Processes multiple Git repositories in a single run.
- **Configurable**: Easily configure repositories, branches, and resource values via environment variables.
- **In-Memory Operations**: Uses an in-memory filesystem for all Git operations, ensuring speed and avoiding disk writes.
- **Extensible Architecture**: Built with a modular design (SOLID principles) that makes it easy to extend and maintain.

## Prerequisites

- Go 1.24.5 or higher

## Configuration

Before running the application, you need to set up your configuration.

1.  Copy the example `.env.example` file to a new file named `.env`:
    ```sh
    cp .env.example .env
    ```

2.  Open the `.env` file and update the variables to match your environment.

| Variable      | Description                                                                                                | Example                               |
|---------------|------------------------------------------------------------------------------------------------------------|---------------------------------------|
| `ENV`         | The target environment name, used to construct the overlay path (e.g., `overlays/<ENV>/...`).                | `development`                         |
| `BASE_URL`    | The base URL of your Git provider. The final repository URL is built as `${BASE_URL}/${REPO_URL}`.             | `https://github.com/your-organization`|
| `BRANCH`      | The branch to clone and commit changes to.                                                                 | `main`                                |
| `REPO_URLS`   | A comma-separated list of repository names to process.                                                     | `my-service-1,my-service-2`           |
| `CPU_REQUEST` | The CPU request to set for the container.                                                                  | `100m`                                |
| `MEM_REQUEST` | The memory request to set for the container.                                                               | `128Mi`                               |
| `CPU_LIMIT`   | The CPU limit to set for the container.                                                                    | `200m`                                |
| `MEM_LIMIT`   | The memory limit to set for the container.                                                                 | `256Mi`                               |
| `GITLAB_BASE_URL`| The base URL of your GitLab instance (defaults to `https://gitlab.com`).                                   | `https://gitlab.yourcompany.com`      |
| `GITLAB_TOKEN`| Your personal GitLab access token (required for the repository fetching script).                           | `your_gitlab_token`                   |
| `GITLAB_GROUP_ID`| The ID of your GitLab group (required for the repository fetching script).                                | `12345`                               |

## Automating Repository Updates

The project includes a script to automatically fetch all repositories from a GitLab group and update the `REPO_URLS` in your `.env` file.

### Script Configuration

To use the script, you need to provide your GitLab token and group ID. You can do this in two ways:

1.  **Environment Variables**: Set the `GITLAB_TOKEN` and `GITLAB_GROUP_ID` environment variables. You can also set `GITLAB_BASE_URL` if you are using a self-hosted GitLab instance.
2.  **`.netrc` File**: Add your GitLab credentials to a `.netrc` file in your home directory. The script will look for a machine that matches the hostname of your `GITLAB_BASE_URL`.

    ```
    machine gitlab.com
      login your_gitlab_username
      password your_gitlab_personal_access_token
    ```

### Running the Script

To run the script, execute the following command:

```sh
make get-repos
```

This will update the `REPO_URLS` in your `.env` file with the latest list of repositories from your GitLab project.

## Usage

To run the main application, which processes the repositories listed in your `.env` file, execute the following command:

```sh
make run
```

The application will then:
1.  Read the configuration from the `.env` file.
2.  Loop through the specified repositories.
3.  Clone each repository into an in-memory filesystem.
4.  Read the `set_resources.yaml` file from the configured overlay path.
5.  Update the resource values.
6.  Commit and push the changes back to the remote repository.

## Development

A `Makefile` is included to streamline common development tasks.

- `make build`: Build the application binary.
- `make test`: Run all tests.
- `make deps`: Tidy and install dependencies.
- `make clean`: Clean up build artifacts.
- `make help`: Display a list of all available targets.

## How It Works

The application is designed with a clean, modular architecture, separating concerns into distinct packages:

- **`cmd/main.go`**: The entry point of the application. It initializes the components and orchestrates the overall workflow.
- **`internal/config`**: Handles loading configuration from the `.env` file.
- **`internal/gitops`**: Manages all Git-related operations, such as cloning, committing, and pushing.
- **`internal/k8s`**: Contains the logic for parsing and patching Kubernetes YAML files. It uses a strategy pattern to easily support different Kubernetes kinds.

## License

This project is licensed under the MIT License.
