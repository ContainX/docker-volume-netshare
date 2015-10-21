package netshare

import (
	"fmt"
	"github.com/calavera/dkvolume"
	"github.com/gondor/docker-volume-netshare/netshare/drivers"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

const (
	UsernameFlag  = "username"
	PasswordFlag  = "password"
	DomainFlag    = "domain"
	VersionFlag   = "version"
	BasedirFlag   = "basedir"
	AvailZoneFlag = "az"
	NoResolveFlag = "noresolve"
	TCPFlag       = "tcp"
	EnvSambaUser  = "NETSHARE_CIFS_USERNAME"
	EnvSambaPass  = "NETSHARE_CIFS_PASSWORD"
	EnvSambaWG    = "NETSHARE_CIFS_DOMAIN"
	PluginAlias   = "netshare"
	NetshareHelp  = `
	docker-volume-netshare (NFS V3/4, AWS EFS and CIFS Volume Driver Plugin)

Provides docker volume support for NFS v3 and 4, EFS as well as CIFS.  This plugin can be run multiple times to
support different mount types.
	`
)

var (
	rootCmd = &cobra.Command{
		Use:   "docker-volume-netshare",
		Short: "NFS and CIFS - Docker volume driver plugin",
		Long:  NetshareHelp,
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
	rootCmd.PersistentFlags().StringVar(&baseDir, BasedirFlag, filepath.Join(dkvolume.DefaultDockerRootDirectory, PluginAlias), "Mounted volume base directory")
	rootCmd.PersistentFlags().String(TCPFlag, ":8877", "Bind to TCP rather than Unix sockets.  :PORT for all interfaces or ADDRESS:PORT to bind")

	cifsCmd.Flags().StringP(UsernameFlag, "u", "", "Username to use for mounts.  Can also set environment NETSHARE_CIFS_USERNAME")
	cifsCmd.Flags().StringP(PasswordFlag, "p", "", "Password to use for mounts.  Can also set environment NETSHARE_CIFS_PASSWORD")
	cifsCmd.Flags().StringP(DomainFlag, "d", "", "Workgroup to use for mounts.  Can also set environment NETSHARE_CIFS_DOMAIN")

	nfsCmd.Flags().IntP(VersionFlag, "v", 4, "NFS Version to use [3 | 4]")

	efsCmd.Flags().String(AvailZoneFlag, "", "AWS Availability zone [default: \"\", looks up via metadata]")
	efsCmd.Flags().Bool(NoResolveFlag, false, "Indicates EFS mount sources are IP Addresses vs File System ID")
}

func execNFS(cmd *cobra.Command, args []string) {
	version, _ := cmd.Flags().GetInt(VersionFlag)
	d := drivers.NewNFSDriver(rootForType(drivers.NFS), version)
	start(drivers.NFS, d)
}

func execEFS(cmd *cobra.Command, args []string) {
	az, _ := cmd.Flags().GetString(AvailZoneFlag)
	resolve, _ := cmd.Flags().GetBool(NoResolveFlag)

	d := drivers.NewEFSDriver(rootForType(drivers.EFS), az, !resolve)
	start(drivers.EFS, d)
}

func execCIFS(cmd *cobra.Command, args []string) {
	user := typeOrEnv(cmd, UsernameFlag, EnvSambaUser)
	pass := typeOrEnv(cmd, PasswordFlag, EnvSambaPass)
	domain := typeOrEnv(cmd, DomainFlag, EnvSambaWG)

	d := drivers.NewCIFSDriver(rootForType(drivers.CIFS), user, pass, domain)
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

func start(dt drivers.DriverType, driver dkvolume.Driver) {
	h := dkvolume.NewHandler(driver)
	fmt.Println(h.ServeUnix("", dt.String()))
}
