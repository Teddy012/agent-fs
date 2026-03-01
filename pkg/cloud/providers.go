package cloud

// ProviderInfo contains information about a cloud storage provider
type ProviderInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Endpoint    string `json:"endpoint_example"`
	ConfigNote  string `json:"config_note,omitempty"`
}

// GetProviders returns a list of all supported cloud storage providers
func GetProviders() []ProviderInfo {
	return []ProviderInfo{
		{
			Name:        "s3",
			Description: "AWS S3",
			Endpoint:    "https://s3.amazonaws.com",
		},
		{
			Name:        "r2",
			Description: "Cloudflare R2",
			Endpoint:    "https://{account_id}.r2.cloudflarestorage.com",
			ConfigNote:  "Set account_id for auto-generated endpoint, or specify endpoint manually",
		},
		{
			Name:        "minio",
			Description: "MinIO Self-Hosted Object Storage",
			Endpoint:    "http://localhost:9000",
			ConfigNote:  "Set path_style=true for non-virtual-hosted-style access",
		},
		{
			Name:        "alioss",
			Description: "Alibaba Cloud Object Storage Service (OSS)",
			Endpoint:    "https://oss-cn-hangzhou.aliyuncs.com",
			ConfigNote:  "Replace region in endpoint: oss-{region}.aliyuncs.com",
		},
		{
			Name:        "txcos",
			Description: "Tencent Cloud Object Storage (COS)",
			Endpoint:    "https://cos.ap-guangzhou.myqcloud.com",
			ConfigNote:  "Replace region in endpoint: cos.{region}.myqcloud.com",
		},
		{
			Name:        "b2",
			Description: "Backblaze B2 (S3 Compatible)",
			Endpoint:    "https://s3.us-west-004.backblazeb2.com",
		},
		{
			Name:        "wasabi",
			Description: "Wasabi Hot Cloud Storage",
			Endpoint:    "https://s3.wasabisys.com",
		},
	}
}
