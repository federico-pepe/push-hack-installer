package pushinstall

import _ "embed"

// A hack bundles everything install.sh's deploy_hack needed for one of
// push-hack's three core hacks, pre-resolved at build time instead of read
// from a cloned push-hack checkout — this installer has no such checkout,
// it only vendors what these three hacks need.
type hack struct {
	id         string
	binaryName string // as in hack.json's "binary" field; empty for none
	binary     []byte
	hackJSON   []byte
	// customInitd is the hack's own service.initd template (only
	// push-display has one — it patches /etc/init.d/push3's LD_PRELOAD line
	// and restarts Push3 on start/stop, which the generic generated script
	// in initd.go knows nothing about). Empty for the other two, which fall
	// back to the generic template, exactly like install.sh's
	// install_hack_service does when no hacks/<id>/service.initd exists.
	customInitd []byte
}

//go:embed vendor/push-manager/push-manager
var pushManagerBinary []byte

//go:embed vendor/push-manager/hack.json
var pushManagerHackJSON []byte

//go:embed vendor/push-catalog/push-catalog
var pushCatalogBinary []byte

//go:embed vendor/push-catalog/hack.json
var pushCatalogHackJSON []byte

//go:embed vendor/push-display/push_hook.so
var pushDisplayBinary []byte

//go:embed vendor/push-display/hack.json
var pushDisplayHackJSON []byte

//go:embed vendor/push-display/service.initd
var pushDisplayServiceInitd []byte

// coreHacks lists the three hacks this installer deploys, always all
// three — the user chose not to build a hack-selection screen, since
// push-hack's own install.sh treats these as the framework's core set.
// Order matters only for display; install/uninstall order doesn't matter
// since each hack is independent.
func coreHacks() []hack {
	return []hack{
		{
			id:         "push-manager",
			binaryName: "push-manager",
			binary:     pushManagerBinary,
			hackJSON:   pushManagerHackJSON,
		},
		{
			id:         "push-catalog",
			binaryName: "push-catalog",
			binary:     pushCatalogBinary,
			hackJSON:   pushCatalogHackJSON,
		},
		{
			id:          "push-display",
			binaryName:  "push_hook.so",
			binary:      pushDisplayBinary,
			hackJSON:    pushDisplayHackJSON,
			customInitd: pushDisplayServiceInitd,
		},
	}
}

// serviceName matches lib/common.sh's service_name(): the init.d script
// name (and systemd unit name, though this port never uses that branch)
// every hack's service is installed under.
func serviceName(hackID string) string {
	return "push-hack-" + hackID
}
