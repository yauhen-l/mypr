# My Pull Requests utility

Checks status of pull requests on Bitbucket server.
It is useful to quickly check status of your pull requests without surfing web interface.
Colors for user status:
GREEN - approved
YELLOW - needs work
CYAN - not approved

## Installation

Build with go and  put binary to your PATH.

Create config, where `user` is your Bitbucket slug:
```
> cat <<EOT >> ~/.config/mypr.yaml
url: "https://yourbitbucket.com"
user: yauhen-l
EOT
```

Btw, you also can add `password: XXXX` section in config, but on your own risk.

## Usage

```
> mypr
Enter Password(yauhenl):
PR-1
        test
        https://yourbitbucket.com/projects/PROJ/repos/test/pull-requests/84
        APPROVED: 5
        UNAPPROVED: 1
          vasa: bad style
            yauhen-l: fixed
```

