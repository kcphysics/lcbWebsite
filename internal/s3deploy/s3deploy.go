package s3deploy

import (
	"context"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// DeploySite deploys the static site to an S3 bucket.
func DeploySite(bucketName, outputDir string) {
	fmt.Printf("Starting deployment to S3 bucket: %s...\n", bucketName)

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS SDK config: %v", err)
	}

	uploader := manager.NewUploader(s3.NewFromConfig(cfg))

	err = filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(outputDir, path)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer file.Close()

		// Determine Content-Type
		contentType := mime.TypeByExtension(filepath.Ext(path))
		if contentType == "" {
			contentType = "application/octet-stream" // Default if type is unknown
		}

		_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
			Bucket:      aws.String(bucketName),
			Key:         aws.String(relPath),
			Body:        file,
			ContentType: aws.String(contentType),
		})
		if err != nil {
			return fmt.Errorf("failed to upload %s to S3: %w", relPath, err)
		}

		fmt.Printf("Uploaded %s to s3://%s/%s\n", relPath, bucketName, relPath)
		return nil
	})

	if err != nil {
		log.Fatalf("Deployment failed: %v", err)
	}

	log.Println("Deployment complete!")
}

// CreateCloudFrontDistribution creates a CloudFront distribution for the given S3 bucket.
func CreateCloudFrontDistribution(bucketName string) (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", fmt.Errorf("failed to load AWS SDK config: %w", err)
	}

	cfClient := cloudfront.NewFromConfig(cfg)

	// Check if a distribution already exists for this bucket
	distID, err := GetCloudFrontDistributionID(bucketName)
	if err != nil {
		return "", fmt.Errorf("failed to check for existing CloudFront distribution: %w", err)
	}
	if distID != "" {
		log.Printf("CloudFront distribution for bucket %s already exists with ID: %s\n", bucketName, distID)
		return distID, nil
	}

	// S3 Origin Domain Name format for CloudFront
	s3OriginDomain := fmt.Sprintf("%s.s3.amazonaws.com", bucketName)

	// Generate a unique caller reference
	callerReference := fmt.Sprintf("aiLCBWebsite-%d", time.Now().Unix())

	input := &cloudfront.CreateDistributionInput{
		DistributionConfig: &types.DistributionConfig{
			CallerReference: aws.String(callerReference),
			Comment:         aws.String(fmt.Sprintf("CloudFront distribution for S3 bucket %s", bucketName)),
			Enabled:         aws.Bool(true),
			DefaultCacheBehavior: &types.DefaultCacheBehavior{
				TargetOriginId:       aws.String(bucketName),
				ViewerProtocolPolicy: types.ViewerProtocolPolicyRedirectToHttps,
				TrustedSigners:       &types.TrustedSigners{Enabled: aws.Bool(false), Quantity: aws.Int32(0)},
				ForwardedValues: &types.ForwardedValues{
					QueryString: aws.Bool(false),
					Cookies:     &types.CookiePreference{Forward: "none"},
				},
				MinTTL: aws.Int64(0),
			},
			Origins: &types.Origins{
				Quantity: aws.Int32(1),
				Items: []types.Origin{
					{
						Id:         aws.String(bucketName),
						DomainName: aws.String(s3OriginDomain),
						S3OriginConfig: &types.S3OriginConfig{
							OriginAccessIdentity: aws.String(""), // No OAI for public S3 bucket
						},
					},
				},
			},
			PriceClass:        types.PriceClassPriceClass100, // US, Europe, Asia, Africa
			DefaultRootObject: aws.String("index.html"),
			Restrictions: &types.Restrictions{
				GeoRestriction: &types.GeoRestriction{
					RestrictionType: types.GeoRestrictionTypeNone,
					Quantity:        aws.Int32(0),
				},
			},
			ViewerCertificate: &types.ViewerCertificate{
				CloudFrontDefaultCertificate: aws.Bool(true),
				MinimumProtocolVersion:       types.MinimumProtocolVersionTLSv12016,
				CertificateSource:            "cloudfront",
			},
		},
	}

	resp, err := cfClient.CreateDistribution(context.TODO(), input)
	if err != nil {
		return "", fmt.Errorf("failed to create CloudFront distribution: %w", err)
	}

	log.Printf("Successfully created CloudFront distribution with ID: %s and Domain Name: %s\n", *resp.Distribution.Id, *resp.Distribution.DomainName)
	return *resp.Distribution.Id, nil
}

// GetCloudFrontDistributionID checks if a CloudFront distribution already exists for the given S3 bucket.
func GetCloudFrontDistributionID(bucketName string) (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", fmt.Errorf("failed to load AWS SDK config: %w", err)
	}

	cfClient := cloudfront.NewFromConfig(cfg)

	paginator := cloudfront.NewListDistributionsPaginator(cfClient, &cloudfront.ListDistributionsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return "", fmt.Errorf("failed to list CloudFront distributions: %w", err)
		}
		for _, dist := range page.DistributionList.Items {
			for _, origin := range dist.Origins.Items {
				// CloudFront origin domain for S3 buckets is bucketName.s3.amazonaws.com
				expectedOriginDomain := fmt.Sprintf("%s.s3.amazonaws.com", bucketName)
				if *origin.DomainName == expectedOriginDomain {
					return *dist.Id, nil
				}
			}
		}
	}
	return "", nil
}
