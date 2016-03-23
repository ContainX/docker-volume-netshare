package netshare

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/gondor/docker-volume-netshare/netshare/drivers"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strconv"
)

const (
	UsernameFlag   = "username"
	PasswordFlag   = "password"
	DomainFlag     = "domain"
	SecurityFlag   = "security"
	VersionFlag    = "version"
	OptionsFlag    = "options"
	BasedirFlag    = "basedir"
	VerboseFlag    = "verbose"
	AvailZoneFlag  = "az"
	NoResolveFlag  = "noresolve"
	NetRCFlag      = "netrc"
	TCPFlag        = "tcp"
	PortFlag       = "port"
	NameServerFlag = "nameserver"
	EnvSambaUser   = "NETSHARE_CIFS_USERNAME"
	EnvSambaPass   = "NETSHARE_CIFS_PASSWORD"
	EnvSambaWG     = "NETSHARE_CIFS_DOMAIN"
	EnvSambaSec    = "NETSHARE_CIFS_SECURITY"
	EnvNfsVers     = "NETSHARE_NFS_VERSION"
	EnvTCP         = "NETSHARE_TCP_ENABLED"
	EnvTCPAddr     = "NETSHARE_TCP_ADDR"
	PluginAlias    = "netshare"
	NetshareHelp   = `
	docker-volume-netshare (NFS V3/4, AWS EFS and CIFS Volume Driver Plugin)

Provides docker volume support for NFS v3 and 4, EFS as well as CIFS.  This plugin can be run multiple times to
support different mount types.
	`
)

var (
	rootCmd = &cobra.Command{
		Use:              "docker-volume-netshare",
		Short:            "NFS and CIFS - Docker volume driver plugin",
		Long:             NetshareHelp,
		PersistentPreRun: setupLogger,
	}

	cifsCmd = &cobra.Command{
		Use:   "cifs",
		Short: "run plugin in CIFS mode",
		Run:   execCIFS,
	}

	nfsCmd = &cobra.Command{
		Use:   "nfs",
		Short: "run plugin in NFS mode",
		Run:   execNFS,
	}

	efsCmd = &cobra.Command{
		Use:   "efs",
		Short: "run plugin in AWS EFS mode",
		Run:   execEFS,
	}
	baseDir = ""
)

func Execute() {
	setupFlags()
	rootCmd.AddCommand(cifsCmd, nfsCmd, efsCmd)
	rootCmd.Execute()
}

func setupFlags() {
	rootCmd.PersistentFlags().StringVar(&baseDir, BasedirFlag, filepath.Join(volume.DefaultDockerRootDirectory, PluginAlias), "Mounted volume base directory")
	rootCmd.PersistentFlags().Bool(TCPFlag, false, "Bind to TCP rather than Unix sockets.  Can also be set via NETSHARE_TCP_ENABLED")
	rootCmd.PersistentFlags().String(PortFlag, ":8877", "TCP Port if --tcp flag is true.  :PORT for all interfaces or ADDRESS:PORT to bind.")
	rootCmd.PersistentFlags().Bool(VerboseFlag, false, "Turns on verbose logging")

	cifsCmd.Flags().StringP(UsernameFlag, "u", "", "Username to use for mounts.  Can also set environment NETSHARE_CIFS_USERNAME")
	cifsCmd.Flags().StringP(PasswordFlag, "p", "", "Password to use for mounts.  Can also set environment NETSHARE_CIFS_PASSWORD")
	cifsCmd.Flags().StringP(DomainFlag, "d", "", "Domain to use for mounts.  Can also set environment NETSHARE_CIFS_DOMAIN")
	cifsCmd.Flags().StringP(SecurityFlag, "s", "", "Security mode to use for mounts (mount.cifs's sec option). Can also set environment NETSHARE_CIFS_SECURITY.")
	cifsCmd.Flags().StringP(NetRCFlag, "", os.Getenv("HOME"), "The default .netrc location.  Default is the user.home directory")

	nfsCmd.Flags().IntP(VersionFlag, "v", 4, "NFS Version to use [3 | 4]. Can also be set with NETSHARE_NFS_VERSION")
	nfsCmd.Flags().StringP(OptionsFlag, "o", "", fmt.Sprintf("Options passed to nfs mounts (ex: %s)", drivers.DefaultNfsV3))

	efsCmd.Flags().String(AvailZoneFlag, "", "AWS Availability zone [default: \"\", looks up via metadata]")
	efsCmd.Flags().String(NameServerFlag, "", "Custom DNS nameserver.  [default \"\", uses /etc/resolv.conf]")
	efsCmd.Flags().Bool(NoResolveFlag, false, "Indicates EFS mount sources are IP Addresses vs File System ID")
}

func setupLogger(cmd *cobra.Command, args []string) {
	if verbose, _ := cmd.Flags().GetBool(VerboseFlag); verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func execNFS(cmd *cobra.Command, args []string) {
	version, _ := cmd.Flags().GetInt(VersionFlag)
	if os.Getenv(EnvNfsVers) != "" {
		if v, err := strconv.Atoi(os.Getenv(EnvNfsVers)); err == nil {
			if v == 3 || v == 4 {
				version = v
			}
		}
	}
	options, _ := cmd.Flags().GetString(OptionsFlag)
	d := drivers.NewNFSDriver(rootForType(drivers.NFS), version, options)
	start(drivers.NFS, d)
}

func execEFS(cmd *cobra.Command, args []string) {
	az, _ := cmd.Flags().GetString(AvailZoneFlag)
	resolve, _ := cmd.Flags().GetBool(NoResolveFlag)
	ns, _ := cmd.Flags().GetString(NameServerFlag)
	d := drivers.NewEFSDriver(rootForType(drivers.EFS), az, ns, !resolve)
	start(drivers.EFS, d)
}

func execCIFS(cmd *cobra.Command, args []string) {
	user := typeOrEnv(cmd, UsernameFlag, EnvSambaUser)
	pass := typeOrEnv(cmd, PasswordFlag, EnvSambaPass)
	domain := typeOrEnv(cmd, DomainFlag, EnvSambaWG)
	security := typeOrEnv(cmd, SecurityFlag, EnvSambaSec)
	netrc, _ := cmd.Flags().GetString(NetRCFlag)

	d := drivers.NewCIFSDriver(rootForType(drivers.CIFS), user, pass, domain, security, netrc)
	start(drivers.CIFS, d)
}

func typeOrEnv(cmd *cobra.Command, flag, envname string) string {
	val, _ := cmd.Flags().GetString(flag)
	if val == "" {
		val = os.Getenv(envname)
	}
	return val
}

func rootForType(dt drivers.DriverType) string {
	return filepath.Join(baseDir, dt.String())
}

func start(dt drivers.DriverType, driver volume.Driver) {
	h := volume.NewHandler(driver)
	if isTCPEnabled() {
		addr := os.Getenv(EnvTCPAddr)
		if addr == "" {
			addr, _ = rootCmd.PersistentFlags().GetString(PortFlag)
		}
		fmt.Println(h.ServeTCP(dt.String(), addr))
	} else {
		fmt.Println(h.ServeUnix("", dt.String()))
	}
}

func isTCPEnabled() bool {
	if tcp, _ := rootCmd.PersistentFlags().GetBool(TCPFlag); tcp {
		return tcp
	}

	if os.Getenv(EnvTCP) != "" {
		ev, _ := strconv.ParseBool(os.Getenv(EnvTCP))
		fmt.Println(ev)

		return ev
	}
	return false
}
