package starter

import "embed"

//go:embed catalog/resources.json web/dist server.cfg cfg/base.cfg cfg/database.cfg cfg/endpoints.cfg cfg/onesync.cfg cfg/permissions.cfg cfg/phone.cfg cfg/resources.cfg cfg/tags.cfg logo.png patches/pma-voice-server-authoritative-calls.patch
var Assets embed.FS
