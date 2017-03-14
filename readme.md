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
url: "https://git.junolab.net"
user: yauhenl
EOT
```

Btw, you also can add `password: XXXX` section in config, but on your own risk.

## Usage

```
> mypr
Enter Password(yauhenl):
BE-6037
        ms_matching
        https://git.junolab.net/projects/MS/repos/ms_matching/pull-requests/84
        APPROVED: 5
        UNAPPROVED: 1
          mkorolyov: seems that ofType methods useless
it can't be inlined by compiler and brings no benefits for code reuse, what do you think?
            yauhenl: I created it to not duplicate error message.
          andrei.shneider: rename _bestETAMatcher_ to _matcher_?
            yauhenl: Not sure.
`BestETA` is actually how `Matcher` is configured. And it's type in database is `BestETA`
```

