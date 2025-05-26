# accio

This project allows you to ask for input from another machine on the same network.

## Why would I do that?

Automation and security are a tradeoff since secrets have to be stored somewhere if you want full automation.
The typical solution is a long running process that stores secrets only in RAM after being unlocked manually once at startup, e.g. ssh-agent, Hashicorp Vault, etc.

For CI jobs, secrets are typically stored in CI variables which have two problems:
 - They rely on the security of your CI provider (GitLab for example is a very complex tool which can have security issues)
 - While those variables are encrypted at rest, the encryption key has to be stored somewhere on the server where it can be extracted (GitLab doesn't require a manual unlock when starting up)

To avoid those problems, there are only two options for CI jobs:
 - use an external secrets store like Hashicorp Vault (only practical for high availability setups, because of the manual unlocking process on startup)
 - ask for manual input during the CI job to ask for an additional secret necessary for the job to run (ssh key passphrase, etc.)

This improved level of security is especially important if you want to run an Ansible playbook in CI, because it requires an ssh key with root access to all your servers.

So if you don't want to manage a high availability setup with Hashicorp Vault, you can use this project to just ask for secrets during the CI job.

## Why not use a jumphost?

You could, but you would need another layer of security so no arbitrary code can be run on the jumphost if an attacker gains access to the ansible repository.
You could for example check the last commit for a valid signature created by an administrator.

While this setup would also be rather secure, I prefer to use a gitlab runner directly, since it also allows me to clearly define what is installed/executed on the runner.
In contrast once a security vulnerability is introduced into a permanent jumpost, it is likely never caught.

## How does it work?

 1. install this project locally on the machines of your administrators and run it as a daemon
 2. download this project inside your CI job (and check its signature)
 3. query a secret with it from the CI job

This tool will then contact a given machine and will show a dialog box on it asking for input.

To run it as a daemon, just run:

```bash
accio
```

To query a machine just run:

```bash
accio -c "ls -alh {{}}" -t "ssh passphrase" -q "Please enter passphrase for ssh private key:"  http://10.0.0.123
```

where `-t` is the title of the dialog box and `-q` is the question in the dialog box shown to the user.
http://10.0.0.123 is the target machine where the dialog box should be shown and with `-c` you can configure the exact command that should be executed using `{{}}` as a placeholder for the secret.
