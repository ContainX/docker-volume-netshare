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
	TypeNFS       = "nfs"
	TypeSMB       = "smb"
	UsernameFlag  = "username"
	PasswordFlag  = "password"
	WorkgroupFlag = "workgroup"
	VersionFlag   = "version"
	BasedirFlag   = "basedir"
	TCPFlag       = "tcp"
	EnvSambaUser  = "NETSHARE_SMB_USERNAME"
	EnvSambaPass  = "NETSHARE_SMB_PASSWORD"
	EnvSambaWG    = "NETSHARE_SMB_WORKGROUP"
	PluginAlias   = "netshare"
	NetshareHelp  = `
	docker-volume-netshare (NFS V3/4, Samba Volume Driver Plugin)

Provides docker volume support for NFS v3 and 4 as well as Samba.  This plugin can be run multiple times to
support different mount types.
	`
)

var (
	rootCmd = &cobra.Command{
		Use:   "docker-volume-netshare",
		Short: "NFS and Samba - Docker volume driver plugin",
		Long:  NetshareHelp,
	}

	sambaCmd = &cobra.Command{
		Use:   "samba",
		Short: "run plugin in Samba mode",
		Run:   execSamba,
	}

	nfsCmd = &cobra.Command{
		Use:   "nfs",
		Short: "run plugin in NFS mode",
		Run:   execNFS,
	}
	baseDir = ""
)

func Execute() {
	setupFlags()
	rootCmd.AddCommand(sambaCmd, nfsCmd)
	rootCmd.Execute()
}

func setupFlags() {
	rootCmd.PersistentFlags().StringVar(&baseDir, BasedirFlag, filepath.Join(dkvolume.DefaultDockerRootDirectory, PluginAlias), "Mounted volume base directory")
	rootCmd.PersistentFlags().String(TCPFlag, ":8877", "Bind to TCP rather than Unix sockets.  :PORT for all interfaces or ADDRESS:PORT to bind")

	sambaCmd.Flags().StringP(UsernameFlag, "u", "", "Username to use for mounts.  Can also set environment NETSHARE_SMB_USERNAME")
	sambaCmd.Flags().StringP(PasswordFlag, "p", "", "Password to use for mounts.  Can also set environment NETSHARE_SMB_PASSWORD")
	sambaCmd.Flags().StringP(WorkgroupFlag, "w", "", "Workgroup to use for mounts.  Can also set environment NETSHARE_SMB_WORKGROUP")
	nfsCmd.Flags().IntP(VersionFlag, "v", 4, "NFS Version to use [3 | 4]")
}

func execNFS(cmd *cobra.Command, args []string) {
	version, _ := cmd.Flags().GetInt(VersionFlag)
	d := drivers.NewNfsDriver(rootForType(TypeNFS), version)
	start(TypeNFS, d)
}

func execSamba(cmd *cobra.Command, args []string) {
	user := typeOrEnv(cmd, UsernameFlag, EnvSambaUser)
	pass := typeOrEnv(cmd, PasswordFlag, EnvSambaPass)
	workgroup := typeOrEnv(cmd, WorkgroupFlag, EnvSambaWG)

	d := drivers.NewSambaDriver(rootForType(TypeSMB), user, pass, workgroup)
	start(TypeSMB, d)

}

func typeOrEnv(cmd *cobra.Command, flag, envname string) string {
	val, _ := cmd.Flags().GetString(flag)
	if val == "" {
		val = os.Getenv(envname)
	}
	return val
}

func rootForType(dt string) string {
	return filepath.Join(baseDir, dt)
}

func start(dt string, driver dkvolume.Driver) {
	h := dkvolume.NewHandler(driver)
	fmt.Println(h.ServeUnix("", dt))
}
