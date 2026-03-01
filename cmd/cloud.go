package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/geekjourneyx/agent-fs/pkg/apperr"
	"github.com/geekjourneyx/agent-fs/pkg/archive"
	"github.com/geekjourneyx/agent-fs/pkg/cloud"
	"github.com/geekjourneyx/agent-fs/pkg/output"
	"github.com/geekjourneyx/agent-fs/pkg/s3client"
	"github.com/geekjourneyx/agent-fs/pkg/sandbox"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cloudProvider  string
	uploadCompress bool

	downloadUnzip     bool
	downloadOverwrite bool

	cloudListLimit int

	cloudURLExpires  int64
	cloudURLPublic   bool
)

var cloudCmd = &cobra.Command{
	Use:   `cloud`,
	Short: `Cloud storage operations`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var cloudUploadCmd = &cobra.Command{
	Use:   `upload <local_path> <remote_key>`,
	Short: `Upload a local file to configured cloud provider`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCloudUpload(args[0], args[1])
	},
}

var cloudDownloadCmd = &cobra.Command{
	Use:   `download <remote_key> <local_path>`,
	Short: `Download a cloud object to local path`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCloudDownload(args[0], args[1])
	},
}

var cloudListCmd = &cobra.Command{
	Use:   `list [prefix]`,
	Short: `List objects in cloud storage`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := ``
		if len(args) > 0 {
			prefix = args[0]
		}
		return runCloudList(prefix)
	},
}

var cloudUrlCmd = &cobra.Command{
	Use:   `url <remote_key>`,
	Short: `Generate access URL for a cloud object`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCloudURL(args[0])
	},
}

var cloudProvidersCmd = &cobra.Command{
	Use:   `providers`,
	Short: `List supported cloud storage providers`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCloudProviders()
	},
}

func init() {
	cloudCmd.AddCommand(cloudUploadCmd)
	cloudCmd.AddCommand(cloudDownloadCmd)
	cloudCmd.AddCommand(cloudListCmd)
	cloudCmd.AddCommand(cloudUrlCmd)
	cloudCmd.AddCommand(cloudProvidersCmd)

	cloudUploadCmd.Flags().StringVar(&cloudProvider, `provider`, ``, `Provider name (s3/r2/minio/alioss/txcos)`)
	cloudUploadCmd.Flags().BoolVar(&uploadCompress, `zip`, false, `Zip source before upload`)

	cloudDownloadCmd.Flags().StringVar(&cloudProvider, `provider`, ``, `Provider name (s3/r2/minio/alioss/txcos)`)
	cloudDownloadCmd.Flags().BoolVar(&downloadUnzip, `unzip`, false, `Unzip downloaded archive into destination directory`)
	cloudDownloadCmd.Flags().BoolVar(&downloadOverwrite, `overwrite`, false, `Overwrite local destination`)

	cloudListCmd.Flags().StringVar(&cloudProvider, `provider`, ``, `Provider name (s3/r2/minio/alioss/txcos)`)
	cloudListCmd.Flags().IntVar(&cloudListLimit, `limit`, 100, `Maximum number of objects to return`)

	cloudUrlCmd.Flags().StringVar(&cloudProvider, `provider`, ``, `Provider name (s3/r2/minio/alioss/txcos)`)
	cloudUrlCmd.Flags().Int64Var(&cloudURLExpires, `expires`, 900, `Expiration time in seconds (default: 900, 15 minutes)`)
	cloudUrlCmd.Flags().BoolVar(&cloudURLPublic, `public`, false, `Generate public URL instead of presigned URL`)
}

func runCloudUpload(localPathArg, remoteKeyArg string) error {
	localPath, err := sandbox.ResolveReadPath(localPathArg)
	if err != nil {
		return err
	}
	if _, err := os.Stat(localPath); err != nil {
		return apperr.Wrap(`cloud_upload`, apperr.CodeNotFound, `local path does not exist`, err)
	}

	uploadPath := localPath
	cleanup := func() {}
	if uploadCompress {
		tempFile, err := os.CreateTemp(``, `afs-upload-*.zip`)
		if err != nil {
			return apperr.Wrap(`cloud_upload`, apperr.CodeArchive, `failed to create temporary archive`, err)
		}
		tempPath := tempFile.Name()
		if err := tempFile.Close(); err != nil {
			return apperr.Wrap(`cloud_upload`, apperr.CodeArchive, `failed to finalize temporary archive path`, err)
		}

		if _, err := archive.Zip(localPath, tempPath); err != nil {
			return err
		}
		uploadPath = tempPath
		cleanup = func() {
			_ = os.Remove(tempPath)
		}
	}
	defer cleanup()

	remoteKey := normalizeRemoteKey(remoteKeyArg, uploadPath)
	providerName := resolveProvider(cloudProvider)

	dispatcher, err := buildCloudDispatcher(providerName)
	if err != nil {
		return err
	}

	result, err := dispatcher.Upload(context.Background(), providerName, cloud.UploadRequest{
		LocalPath: uploadPath,
		RemoteKey: remoteKey,
	})
	if err != nil {
		return err
	}
	result.Compressed = uploadCompress

	if err := output.PrintSuccess(`cloud_upload`, result); err != nil {
		return apperr.Wrap(`cloud_upload`, apperr.CodeInternal, `failed to write output`, err)
	}
	return nil
}

func runCloudDownload(remoteKeyArg, localPathArg string) error {
	providerName := resolveProvider(cloudProvider)
	dispatcher, err := buildCloudDispatcher(providerName)
	if err != nil {
		return err
	}

	remoteKey := strings.TrimSpace(remoteKeyArg)
	if remoteKey == `` {
		return apperr.New(`cloud_download`, apperr.CodeInvalidArg, `remote_key is required`)
	}

	targetPath := localPathArg
	var (
		finalPath = localPathArg
		cleanup   = func() {}
	)

	if downloadUnzip {
		destDir, err := sandbox.ResolveWritePath(localPathArg)
		if err != nil {
			return err
		}
		finalPath = destDir
		if !downloadOverwrite && destinationExists(destDir) {
			return apperr.New(`cloud_download`, apperr.CodeConflict, `destination already exists; use --overwrite`)
		}
		tempFile, err := os.CreateTemp(``, `afs-download-*.zip`)
		if err != nil {
			return apperr.Wrap(`cloud_download`, apperr.CodeArchive, `failed to create temporary file`, err)
		}
		targetPath = tempFile.Name()
		if err := tempFile.Close(); err != nil {
			return apperr.Wrap(`cloud_download`, apperr.CodeArchive, `failed to finalize temporary path`, err)
		}
		cleanup = func() {
			_ = os.Remove(targetPath)
		}
	} else {
		resolved, err := sandbox.ResolveWritePath(localPathArg)
		if err != nil {
			return err
		}
		finalPath = resolved
		targetPath = resolved
	}
	defer cleanup()

	if !downloadOverwrite {
		if _, err := os.Stat(targetPath); err == nil {
			return apperr.New(`cloud_download`, apperr.CodeConflict, `destination already exists; use --overwrite`)
		}
	}

	result, err := dispatcher.Download(context.Background(), providerName, cloud.DownloadRequest{
		RemoteKey: remoteKey,
		LocalPath: targetPath,
	})
	if err != nil {
		return err
	}

	if downloadUnzip {
		files, bytes, err := archive.Unzip(targetPath, finalPath)
		if err != nil {
			return err
		}
		result.LocalPath = finalPath
		result.Decompressed = true
		result.ExtractedFiles = files
		result.ExtractedBytes = bytes
	}

	if err := output.PrintSuccess(`cloud_download`, result); err != nil {
		return apperr.Wrap(`cloud_download`, apperr.CodeInternal, `failed to write output`, err)
	}
	return nil
}

func buildCloudDispatcher(providerName string) (*cloud.Dispatcher, error) {
	cfg, err := loadS3ProviderConfig(providerName)
	if err != nil {
		return nil, err
	}
	provider, err := cloud.NewS3CompatibleProvider(providerName, cfg)
	if err != nil {
		return nil, err
	}
	return cloud.NewDispatcher(map[string]cloud.Provider{
		strings.ToLower(providerName): provider,
	}), nil
}

func loadS3ProviderConfig(providerName string) (s3client.Config, error) {
	prefixes := []string{
		fmt.Sprintf(`providers.%s`, strings.ToLower(providerName)),
		strings.ToLower(providerName),
		`providers.s3`,
		`s3`,
	}

	cfg := s3client.Config{
		Endpoint:        pickString(prefixes, `endpoint`),
		Region:          pickString(prefixes, `region`),
		Bucket:          pickString(prefixes, `bucket`),
		AccessKeyID:     pickString(prefixes, `access_key_id`, `accesskeyid`, `access_key`, `accesskey`),
		SecretAccessKey: pickString(prefixes, `secret_access_key`, `access_key_secret`, `secretkey`, `secret_key`),
		PathPrefix:      pickString(prefixes, `path`, `path_prefix`),
		CDNHost:         pickString(prefixes, `cdn_host`, `domain`),
		PathStyle:       pickBool(prefixes, false, `path_style`, `pathstyle`),
		UseSSL:          pickBool(prefixes, true, `use_ssl`),
	}

	if strings.EqualFold(providerName, `r2`) && cfg.Endpoint == `` {
		accountID := pickString(prefixes, `account_id`, `accountid`)
		if accountID != `` {
			cfg.Endpoint = fmt.Sprintf(`https://%s.r2.cloudflarestorage.com`, accountID)
		}
	}
	if cfg.Bucket == `` {
		return s3client.Config{}, apperr.New(`cloud_upload`, apperr.CodeConfig, `missing provider bucket configuration`)
	}
	if cfg.AccessKeyID == `` || cfg.SecretAccessKey == `` {
		return s3client.Config{}, apperr.New(`cloud_upload`, apperr.CodeConfig, `missing provider access key configuration`)
	}
	return cfg, nil
}

func resolveProvider(flagValue string) string {
	if value := strings.TrimSpace(flagValue); value != `` {
		return strings.ToLower(value)
	}
	if value := strings.TrimSpace(viper.GetString(`cloud.provider`)); value != `` {
		return strings.ToLower(value)
	}
	return `s3`
}

func normalizeRemoteKey(remoteKeyArg, localPath string) string {
	key := strings.TrimSpace(remoteKeyArg)
	if strings.HasSuffix(key, `/`) {
		return key + filepath.Base(localPath)
	}
	return key
}

func pickString(prefixes []string, keys ...string) string {
	for _, prefix := range prefixes {
		for _, key := range keys {
			fullKey := prefix + `.` + key
			if value := strings.TrimSpace(viper.GetString(fullKey)); value != `` {
				return value
			}
		}
	}
	return ``
}

func pickBool(prefixes []string, defaultValue bool, keys ...string) bool {
	for _, prefix := range prefixes {
		for _, key := range keys {
			fullKey := prefix + `.` + key
			if viper.IsSet(fullKey) {
				return viper.GetBool(fullKey)
			}
		}
	}
	return defaultValue
}

func destinationExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func runCloudList(prefixArg string) error {
	providerName := resolveProvider(cloudProvider)
	dispatcher, err := buildCloudDispatcher(providerName)
	if err != nil {
		return err
	}

	prefix := strings.TrimSpace(prefixArg)
	limit := cloudListLimit
	if limit <= 0 {
		limit = 100
	}

	result, err := dispatcher.List(context.Background(), providerName, cloud.ListRequest{
		Prefix: prefix,
		Limit:  limit,
	})
	if err != nil {
		return err
	}

	return output.PrintSuccess(`cloud_list`, map[string]any{
		"provider":     result.Provider,
		"objects":      result.Objects,
		"count":        result.Count,
		"total_bytes":  result.TotalBytes,
		"prefix":       result.Prefix,
		"is_truncated": result.IsTruncated,
	})
}

func runCloudURL(remoteKeyArg string) error {
	providerName := resolveProvider(cloudProvider)
	dispatcher, err := buildCloudDispatcher(providerName)
	if err != nil {
		return err
	}

	remoteKey := strings.TrimSpace(remoteKeyArg)
	if remoteKey == `` {
		return apperr.New(`cloud_url`, apperr.CodeInvalidArg, `remote_key is required`)
	}

	// Warn user about public URL security implications
	if cloudURLPublic {
		warnPublicURL(providerName)
	}

	result, err := dispatcher.URL(context.Background(), providerName, cloud.URLRequest{
		RemoteKey:  remoteKey,
		Expiration: cloudURLExpires,
		PublicOnly: cloudURLPublic,
	})
	if err != nil {
		return err
	}

	return output.PrintSuccess(`cloud_url`, map[string]any{
		"provider":     result.Provider,
		"remote_key":    result.RemoteKey,
		"url":           result.URL,
		"expires_in":    result.ExpiresIn,
		"expires_at":    result.ExpiresAt,
		"is_presigned":  result.IsPresigned,
	})
}

func warnPublicURL(provider string) {
	msg := `
⚠️  SECURITY WARNING: You are using --public flag

The generated URL will be publicly accessible without authentication.
This is only suitable for:
  • Public website assets (images, CSS, JS)
  • Public download files
  • Files intended for public sharing

DO NOT use --public for:
  • Private configuration files
  • Sensitive data or logs
  • Any files requiring access control

For Cloudflare R2: Public Access must be enabled in your bucket settings.
`
	fmt.Fprint(os.Stderr, msg)
}

func runCloudProviders() error {
	providers := cloud.GetProviders()

	providerList := make([]map[string]any, len(providers))
	for i, p := range providers {
		providerList[i] = map[string]any{
			"name":             p.Name,
			"description":      p.Description,
			"endpoint_example": p.Endpoint,
		}
		if p.ConfigNote != "" {
			providerList[i]["config_note"] = p.ConfigNote
		}
	}

	return output.PrintSuccess(`cloud_providers`, map[string]any{
		"providers": providerList,
		"note":      "Any S3-compatible storage is supported. Configure custom endpoint via config.",
	})
}
