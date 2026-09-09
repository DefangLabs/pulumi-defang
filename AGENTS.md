# Agent guidance for pulumi-defang

See CLAUDE.md for build/test/architecture details. This file collects
standing guidance from maintainers that should shape how agents write code
here, independent of any one PR or task.

## Don't mirror cloud-provider validation

Don't pre-validate values that the underlying cloud API already validates
(port ranges, reserved values, resource limits, naming rules, etc.). That
duplicated logic drifts as the cloud's own rules change, and it's one more
place to get subtly wrong or out of date.

Instead, let the cloud API reject the bad input, and make sure that failure
bubbles up through the provider in a way that's clear and actionable for
whoever is deploying — a human or an agent reading the error should be able
to tell what was wrong and how to fix it, without needing to read this
codebase's source to decode it.

This came up on PR #582 (Azure host-mode-port ingress): the initial fix
added explicit checks for UDP host ports, Azure's reserved port 36985, and
ports 80/443, duplicating validation Azure Container Apps' ARM API already
performs. Removed in favor of letting the ARM error surface directly.
