package cmd

import (
	"os"

	"github.com/geekjourneyx/agent-fs/pkg/apperr"
	"github.com/geekjourneyx/agent-fs/pkg/archive"
	"github.com/geekjourneyx/agent-fs/pkg/local"
	"github.com/geekjourneyx/agent-fs/pkg/output"
	"github.com/geekjourneyx/agent-fs/pkg/sandbox"
	"github.com/spf13/cobra"
)

var (
	localZipOut      string
	localUnzipTo     string
	localInfoDetails bool
	localReadHead    int64
	localReadTail    int64
	localReadBytes   int64
)

var localCmd = &cobra.Command{
	Use:   `local`,
	Short: `Local filesystem operations`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var localZipCmd = &cobra.Command{
	Use:   `zip <source_path>`,
	Short: `Create zip archive from local file or directory`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLocalZip(args[0])
	},
}

var localUnzipCmd = &cobra.Command{
	Use:   `unzip <zip_path>`,
	Short: `Extract zip archive to destination directory`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLocalUnzip(args[0])
	},
}

var localInfoCmd = &cobra.Command{
	Use:   `info <path>`,
	Short: `Get file or directory metadata`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLocalInfo(args[0])
	},
}

var localReadCmd = &cobra.Command{
	Use:   `read <path>`,
	Short: `Read file content with slicing options`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLocalRead(args[0])
	},
}

func init() {
	localCmd.AddCommand(localZipCmd)
	localCmd.AddCommand(localUnzipCmd)
	localCmd.AddCommand(localInfoCmd)
	localCmd.AddCommand(localReadCmd)

	localZipCmd.Flags().StringVar(&localZipOut, `out`, ``, `Output zip path`)
	_ = localZipCmd.MarkFlagRequired(`out`)

	localUnzipCmd.Flags().StringVar(&localUnzipTo, `dest`, ``, `Destination directory`)
	_ = localUnzipCmd.MarkFlagRequired(`dest`)

	localInfoCmd.Flags().BoolVar(&localInfoDetails, `details`, false, `Include file count and total bytes for directories`)

	localReadCmd.Flags().Int64Var(&localReadHead, `head`, 0, `Read first N lines`)
	localReadCmd.Flags().Int64Var(&localReadTail, `tail`, 0, `Read last N lines`)
	localReadCmd.Flags().Int64Var(&localReadBytes, `bytes`, 0, `Read first N bytes`)
}

func runLocalZip(sourceArg string) error {
	sourcePath, err := sandbox.ResolveReadPath(sourceArg)
	if err != nil {
		return err
	}
	if _, err := os.Stat(sourcePath); err != nil {
		return apperr.Wrap(`local_zip`, apperr.CodeNotFound, `source path does not exist`, err)
	}

	outputPath, err := sandbox.ResolveWritePath(localZipOut)
	if err != nil {
		return err
	}
	size, err := archive.Zip(sourcePath, outputPath)
	if err != nil {
		return err
	}

	return output.PrintSuccess(`local_zip`, map[string]any{
		"source_path": sourcePath,
		"zip_path":    outputPath,
		"size_bytes":  size,
	})
}

func runLocalUnzip(zipArg string) error {
	zipPath, err := sandbox.ResolveReadPath(zipArg)
	if err != nil {
		return err
	}
	if _, err := os.Stat(zipPath); err != nil {
		return apperr.Wrap(`local_unzip`, apperr.CodeNotFound, `zip file does not exist`, err)
	}

	destPath, err := sandbox.ResolveWritePath(localUnzipTo)
	if err != nil {
		return err
	}
	files, size, err := archive.Unzip(zipPath, destPath)
	if err != nil {
		return err
	}

	return output.PrintSuccess(`local_unzip`, map[string]any{
		"zip_path":        zipPath,
		"destination":     destPath,
		"extracted_files": files,
		"size_bytes":      size,
	})
}

func runLocalInfo(pathArg string) error {
	resolvedPath, err := sandbox.ResolveReadPath(pathArg)
	if err != nil {
		return err
	}

	fileInfo, dirInfo, err := local.GetInfo(resolvedPath, localInfoDetails)
	if err != nil {
		return err
	}

	// For directories with details, return directory info
	if fileInfo.IsDir && localInfoDetails {
		return output.PrintSuccess(`local_info`, map[string]any{
			"path":          dirInfo.Path,
			"type":          "directory",
			"file_count":    dirInfo.FileCount,
			"total_bytes":   dirInfo.TotalBytes,
			"mode":          dirInfo.Mode,
			"modified_time": dirInfo.ModifiedTime,
		})
	}

	// For files or directories without details, return file info
	return output.PrintSuccess(`local_info`, map[string]any{
		"name":          fileInfo.Name,
		"path":          fileInfo.Path,
		"type":          fileInfo.Type,
		"size_bytes":    fileInfo.SizeBytes,
		"mode":          fileInfo.Mode,
		"modified_time": fileInfo.ModifiedTime,
		"is_dir":        fileInfo.IsDir,
	})
}

func runLocalRead(pathArg string) error {
	resolvedPath, err := sandbox.ResolveReadPath(pathArg)
	if err != nil {
		return err
	}

	result, err := local.ReadFile(resolvedPath, local.ReadOptions{
		HeadLines: localReadHead,
		TailLines: localReadTail,
		Bytes:     localReadBytes,
	})
	if err != nil {
		return err
	}

	return output.PrintSuccess(`local_read`, map[string]any{
		"path":       result.Path,
		"content":    result.Content,
		"line_count": result.LineCount,
		"byte_count": result.ByteCount,
		"truncated":  result.Truncated,
		"slice_type": result.SliceType,
	})
}
