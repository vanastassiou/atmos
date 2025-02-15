---
title: atmos describe affected
sidebar_label: affected
sidebar_class_name: command
id: affected
description: This command produces a list of the affected Atmos components and stacks given two Git commits.
---

:::note Purpose
Use this command to show a list of the affected Atmos components and stacks given two Git commits.
:::

## Description

The command uses two different Git commits to produce a list of affected Atmos components and stacks.

For the first commit, the command assumes that the current repo root is a Git checkout. An error will be thrown if the current repo is not a Git
repository (`.git` folder does not exist or is configured incorrectly).

The second commit is specified on the command line by using
the `--ref` ([Git References](https://git-scm.com/book/en/v2/Git-Internals-Git-References)) or `--sha` (commit SHA) flags.

Either `--ref` or `--sha` should be used. If both flags are provided at the same time, the command will first clone the remote branch pointed to by
the `--ref` flag and then checkout the Git commit pointed to by the `--sha` flag (`--sha` flag overrides `--ref` flag).

If the flags are not provided, the `ref` will be set automatically to the reference to the default branch (e.g. `main`) and the commit SHA will point
to the `HEAD` of the branch.

Note that if you specify the `--repo-path` flag with the path to the already cloned repository, the command will not clone the target
repository, but instead will use the already cloned one to compare the current branch with. In this case, the `--ref`, `--sha`, `--ssh-key`
and `--ssh-key-password` flags are not used, and an error will be thrown if the `--repo-path` flag and any of the `--ref`, `--sha`, `--ssh-key`
or `--ssh-key-password` flags are provided at the same time.

The command works by:

- Cloning the target branch (`--ref`) or checking out the commit (`--sha`) of the remote target branch, or using the already cloned target repository
  specified by the `--repo-path` flag
- Deep merging all stack configurations for both the current working branch and the target branch
- Looking for changes in the component directories
- Comparing each section of the stack configuration looking for differences
- Output a JSON or YAML document consisting of a list of affected components and stacks and what caused it to be affected

Since Atmos first checks the component folders for changes, if it finds any affected files, it will mark all related components and stacks as
affected. Atmos will then skip evaluating those stacks for differences since we already know that they are affected.

<br/>

```shell
> atmos describe affected --verbose=true

Cloning repo 'https://github.com/cloudposse/atmos' into the temp dir '/var/folders/g5/lbvzy_ld2hx4mgrgyp19bvb00000gn/T/16710736261366892599'

Checking out the HEAD of the default branch ...

Enumerating objects: 4215, done.
Counting objects: 100% (1157/1157), done.
Compressing objects: 100% (576/576), done.
Total 4215 (delta 658), reused 911 (delta 511), pack-reused 3058

Checked out Git ref 'refs/heads/master'

Current working repo HEAD: 7d37c1e890514479fae404d13841a2754be70cbf refs/heads/describe-affected
Remote repo HEAD: 40210e8d365d3d88ac13c0778c0867b679bbba69 refs/heads/master

Changed files:

examples/complete/components/terraform/infra/vpc/main.tf
internal/exec/describe_affected.go
website/docs/cli/commands/describe/describe-affected.md

Affected components and stacks:

[
   {
      "component": "infra/vpc",
      "component_type": "terraform",
      "stack": "tenant1-ue2-dev",
      "spacelift_stack": "tenant1-ue2-dev-infra-vpc",
      "affected": "component"
   },
   {
      "component": "infra/vpc",
      "component_type": "terraform",
      "stack": "tenant1-ue2-prod",
      "spacelift_stack": "tenant1-ue2-prod-infra-vpc",
      "affected": "component"
   },
   {
      "component": "infra/vpc",
      "component_type": "terraform",
      "stack": "tenant1-ue2-staging",
      "spacelift_stack": "tenant1-ue2-staging-infra-vpc",
      "affected": "component"
   }
]
```

<br/>

## Usage

```shell
atmos describe affected [options]
```

<br/>

:::tip
Run `atmos describe affected --help` to see all the available options
:::

## Examples

```shell
atmos describe affected
atmos describe affected --verbose=true
atmos describe affected --ref refs/heads/main
atmos describe affected --ref refs/heads/my-new-branch --verbose=true
atmos describe affected --ref refs/heads/main --format json
atmos describe affected --ref refs/tags/v1.16.0 --file affected.yaml --format yaml
atmos describe affected --sha 3a5eafeab90426bd82bf5899896b28cc0bab3073 --file affected.json
atmos describe affected --sha 3a5eafeab90426bd82bf5899896b28cc0bab3073
atmos describe affected --ssh-key <path_to_ssh_key>
atmos describe affected --ssh-key <path_to_ssh_key> --ssh-key-password <password>
atmos describe affected --repo-path <path_to_already_cloned_repo>
```

## Flags

| Flag                 | Description                                                                                                                                                      | Required |
|:---------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------|:---------|
| `--ref`              | [Git Reference](https://git-scm.com/book/en/v2/Git-Internals-Git-References) with which to compare the current working branch                                    | no       |
| `--sha`              | Git commit SHA with which to compare the current working branch                                                                                                  | no       |
| `--file`             | If specified, write the result to the file                                                                                                                       | no       |
| `--format`           | Specify the output format: `json` or `yaml` (`json` is default)                                                                                                  | no       |
| `--verbose`          | Print more detailed output when cloning and checking out the Git repository<br/>and processing the result                                                        | no       |
| `--ssh-key`          | Path to PEM-encoded private key to clone private repos using SSH                                                                                                 | no       |
| `--ssh-key-password` | Encryption password for the PEM-encoded private key if the key contains<br/>a password-encrypted PEM block                                                       | no       |
| `--repo-path`        | Path to the already cloned target repository with which to compare the current branch.<br/>Conflicts with `--ref`, `--sha`, `--ssh-key` and `--ssh-key-password` | no       |

## Output

The command outputs a list of objects (in JSON or YAML format).

Each object has the following schema:

```json
{
  "component": "....",
  "component_type": "....",
  "stack": "....",
  "spacelift_stack": ".....",
  "affected": "....."
}
```

where:

- `component` - the affected Atmos component in the stack
- `component_type` - the type of the component (`terraform` or `helmfile`)
- `stack` - the affected Atmos stack
- `spacelift_stack` - the affected Spacelift stack. It will be included only if the Spacelift workspace is enabled for the Atmos component in the
  Atmos stack in the `settings.spacelift.workspace_enabled` config
- `affected` - shows what was changed for the component. The possible values are:

  - `stack.vars` - the `vars` component section in the stack config has been modified
  - `stack.env` - the `env` component section in the stack config has been modified
  - `stack.settings` - the `settings` component section in the stack config has been modified
  - `stack.metadata` - the `metadata` component section in the stack config has been modified
  - `component` - the Terraform or Helmfile component that the Atmos component provisions has been changed

<br/>

For example:

```json
[
  {
    "component": "test/test-component-override-2",
    "component_type": "terraform",
    "stack": "tenant1-ue2-dev",
    "spacelift_stack": "tenant1-ue2-dev-new-component",
    "affected": "stack.vars"
  },
  {
    "component": "infra/vpc",
    "component_type": "terraform",
    "stack": "tenant2-ue2-staging",
    "spacelift_stack": "tenant1-ue2-staging-infra-vpc",
    "affected": "component"
  },
  {
    "component": "test/test-component-override-3",
    "component_type": "terraform",
    "stack": "tenant1-ue2-prod",
    "affected": "stack.env"
  }
]
```

<br/>

## Working with Private Repositories

There are a few ways to work with private repositories with which the current local branch is compared to detect the changed files and affected Atmos
stacks and components:

- Using the `--ssh-key` flag to specify the filesystem path to a PEM-encoded private key to clone private repos using SSH, and
  the `--ssh-key-password` flag to provide the encryption password for the PEM-encoded private key if the key contains a password-encrypted PEM block

- Execute the `atmos describe affected` command in a [GitHub Action](https://docs.github.com/en/actions), clone the remote target repository in the
  action, and use the `--repo-path` flag to specify the path to the already cloned target repository with which to compare the current branch

## Using with GitHub Actions

If the `atmos describe affected` command is executed in a [GitHub Action](https://docs.github.com/en/actions), and you don't want to store or
generate a long-lived SSH private key on the server, you can do the following:

- Create a GitHub
  [Personal Access Token (PAT)](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
  with scope permissions to clone private repos

- Add the created PAT as a repository or GitHub organization [secret](https://docs.github.com/en/actions/security-guides/encrypted-secrets) with the
  name [`GITHUB_TOKEN`](https://docs.github.com/en/actions/security-guides/automatic-token-authentication)

- In your GitHub action, clone the remote repository using the [checkout](https://github.com/actions/checkout) GitHub action

- Execute `atmos describe affected` command with the `--repo-path` flag set to the cloned repository path using
  the [`GITHUB_WORKSPACE`](https://docs.github.com/en/actions/learn-github-actions/variables) ENV variable (which points to the default working
  directory on the GitHub runner for steps, and the default location of the repository when using the [checkout](https://github.com/actions/checkout)
  action). For example:

    ```shell
    atmos describe affected --repo-path $GITHUB_WORKSPACE
    ```
