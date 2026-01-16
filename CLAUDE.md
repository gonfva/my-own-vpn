The app is a VPN client. Please review @README.md for a bit more detail.

Initially you were going to be the architect and a different AI the implementer. You created the tickets in Github.

However, you're now the implementer. We will be picking up tickets and creating PRs. Use judgement when implementing those tickets. Maybe your approach should change from what appears in the ticket.

Before starting a new piece of functionality, create a new branch from master.

Creeate a reasonable set of tests. There is no need for 100% coverage, though.

Before creating a PR, make sure that the code is linted (golangci-lint) and tests pass.

The code will be open source, so have security in mind.

If you ask clarification questions, it might be good to modify either this file or README.md
