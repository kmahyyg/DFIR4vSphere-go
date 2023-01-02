# DFIR4vSphere-go

DFIR4vSphere Golang Implementation [WIP]

# Build and Compile

Build:

```shell
go mod download
go mod tidy
./scripts/build.sh
```

# Usage

Check [docs](/docs/get-started.md) .

For detailed commands: Check [full help docs](/docs/full-help.md).

Please only use `exit` command for exit, if not necessary, do not use Ctrl-C to exit directly.

# Debug

Try: https://github.com/vmware/govmomi/tree/main/vcsim which mocked a vCenter API.

# Privacy

Your usage data will be collected for usage telemetry if you use prebuilt binary.

All data are stored in database hosted on Cloudflare, 
those data will only be used for usage stats, and will not disclose to any third-party.

To opt-out from telemetry: `export IW0ulDL1Ke2OPT0UtFr0MTeLEmETrY=1`

# License

GNU AGPL v3

Additionally:
- You are not allowed to use this program for any commercial purpose without explicit purchase of commercial license.

To purchase commercial license, contact me using [Telegram](https://t.me/GH_Consult_7c88e09_bot) .

All purchase will be divided into 3 parts, 30% for my daily life, 50% to original project authors, 20% donated to Red Cross.

All purchase history will be published in GitHub Wiki specific page.

中国大陆用户: 中国大陆司法实践中承认 AGPLv3/GPLv3 协议具备合同特征，属于民事法律行为，属于《中华人民共和国合同法》调整的范围。

# Original Project (Credit!)

- https://github.com/ANSSI-FR/DFIR4vSphere Licensed under GPLv3.

This project is a GoLang-implementation of above projects, and it cannot be made without their success.