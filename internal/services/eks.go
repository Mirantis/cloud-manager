package services

import (
	"fmt"
	"sync"

	"github.com/rusik69/aws-iam-manager/internal/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
)

func (s *AWSService) ListEKSClusters() ([]models.EKSCluster, error) {
	const cacheKey = "eks-clusters"

	if cached, found := s.cache.Get(cacheKey); found {
		if clusters, ok := cached.([]models.EKSCluster); ok {
			return clusters, nil
		}
	}

	accounts, err := s.ListAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %v", err)
	}

	var accessibleAccounts []models.Account
	for _, account := range accounts {
		if account.Accessible {
			accessibleAccounts = append(accessibleAccounts, account)
		}
	}

	if len(accessibleAccounts) == 0 {
		return []models.EKSCluster{}, nil
	}

	type accountResult struct {
		clusters  []models.EKSCluster
		err       error
		accountID string
	}

	resultChan := make(chan accountResult, len(accessibleAccounts))
	var wg sync.WaitGroup

	for _, account := range accessibleAccounts {
		wg.Add(1)
		go func(acc models.Account) {
			defer wg.Done()
			clusters, err := s.getEKSClustersForAccount(acc)
			resultChan <- accountResult{
				clusters:  clusters,
				err:       err,
				accountID: acc.ID,
			}
		}(account)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var allClusters []models.EKSCluster
	for result := range resultChan {
		if result.err != nil {
			fmt.Printf("[WARNING] Failed to get EKS clusters for account %s: %v\n", result.accountID, result.err)
			continue
		}
		allClusters = append(allClusters, result.clusters...)
	}

	s.cache.Set(cacheKey, allClusters, s.cacheTTL)

	return allClusters, nil
}

func (s *AWSService) getEKSClustersForAccount(account models.Account) ([]models.EKSCluster, error) {
	sess, err := s.getSessionForAccount(account.ID)
	if err != nil {
		return nil, fmt.Errorf("cannot access account %s: %w", account.ID, err)
	}

	regions, err := s.getAccessibleRegions(sess)
	if err != nil || len(regions) == 0 {
		return []models.EKSCluster{}, nil
	}

	type regionResult struct {
		clusters []models.EKSCluster
		region   string
	}

	resultChan := make(chan regionResult, len(regions))
	var wg sync.WaitGroup

	for _, region := range regions {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			regionSess := sess.Copy(&aws.Config{Region: aws.String(r)})
			clusters, err := s.getEKSClustersForRegion(regionSess, account, r)
			if err != nil {
				fmt.Printf("[WARNING] Failed to get EKS clusters in region %s for account %s: %v\n", r, account.ID, err)
				clusters = []models.EKSCluster{}
			}
			resultChan <- regionResult{clusters: clusters, region: r}
		}(region)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var allClusters []models.EKSCluster
	for result := range resultChan {
		allClusters = append(allClusters, result.clusters...)
	}

	return allClusters, nil
}

func (s *AWSService) getEKSClustersForRegion(sess *session.Session, account models.Account, region string) ([]models.EKSCluster, error) {
	eksClient := eks.New(sess)

	input := &eks.ListClustersInput{}
	result, err := eksClient.ListClusters(input)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %v", err)
	}

	if len(result.Clusters) == 0 {
		return []models.EKSCluster{}, nil
	}

	type clusterResult struct {
		cluster models.EKSCluster
		err     error
	}

	resultChan := make(chan clusterResult, len(result.Clusters))
	var wg sync.WaitGroup

	for _, clusterName := range result.Clusters {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			cluster, err := s.describeCluster(sess, account, region, name)
			if err != nil {
				fmt.Printf("[WARNING] Failed to describe cluster %s: %v\n", name, err)
				resultChan <- clusterResult{err: err}
				return
			}
			resultChan <- clusterResult{cluster: cluster}
		}(*clusterName)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var clusters []models.EKSCluster
	for result := range resultChan {
		if result.err == nil {
			clusters = append(clusters, result.cluster)
		}
	}

	return clusters, nil
}

func (s *AWSService) describeCluster(sess *session.Session, account models.Account, region, clusterName string) (models.EKSCluster, error) {
	eksClient := eks.New(sess)

	output, err := eksClient.DescribeCluster(&eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})
	if err != nil {
		return models.EKSCluster{}, fmt.Errorf("failed to describe cluster: %v", err)
	}

	cluster := output.Cluster
	authMode := ""
	if cluster.AccessConfig != nil {
		authMode = aws.StringValue(cluster.AccessConfig.AuthenticationMode)
	}

	tags := make([]models.Tag, 0)
	if cluster.Tags != nil {
		for k, v := range cluster.Tags {
			tags = append(tags, models.Tag{Key: k, Value: aws.StringValue(v)})
		}
	}

	clusterModel := models.EKSCluster{
		ClusterName:     aws.StringValue(cluster.Name),
		ClusterARN:      aws.StringValue(cluster.Arn),
		AccountID:       account.ID,
		AccountName:     account.Name,
		Region:          region,
		Status:          aws.StringValue(cluster.Status),
		PlatformVersion: aws.StringValue(cluster.PlatformVersion),
		Arn:             aws.StringValue(cluster.Arn),
		Endpoint:        aws.StringValue(cluster.Endpoint),
		AuthMode:        authMode,
		Tags:            tags,
	}

	if cluster.CreatedAt != nil {
		clusterModel.CreatedAt = cluster.CreatedAt
	}

	// Fetch node groups
	ngResult, err := eksClient.ListNodegroups(&eks.ListNodegroupsInput{
		ClusterName: aws.String(clusterName),
	})
	if err == nil && len(ngResult.Nodegroups) > 0 {
		var nodeGroups []models.NodeGroup
		for _, ngName := range ngResult.Nodegroups {
			ngDesc, err := eksClient.DescribeNodegroup(&eks.DescribeNodegroupInput{
				ClusterName:   aws.String(clusterName),
				NodegroupName: ngName,
			})
			if err != nil {
				continue
			}
			ng := ngDesc.Nodegroup
			scaling := models.NodeScaling{}
			if ng.ScalingConfig != nil {
				scaling = models.NodeScaling{
					DesiredSize: aws.Int64Value(ng.ScalingConfig.DesiredSize),
					MinSize:     aws.Int64Value(ng.ScalingConfig.MinSize),
					MaxSize:     aws.Int64Value(ng.ScalingConfig.MaxSize),
				}
			}
			ngTags := make([]models.Tag, 0)
			if ng.Tags != nil {
				for k, v := range ng.Tags {
					ngTags = append(ngTags, models.Tag{Key: k, Value: aws.StringValue(v)})
				}
			}
			labels := make(map[string]string)
			if ng.Labels != nil {
				for k, v := range ng.Labels {
					labels[k] = aws.StringValue(v)
				}
			}
			instanceTypes := make([]string, 0, len(ng.InstanceTypes))
			for _, it := range ng.InstanceTypes {
				if it != nil {
					instanceTypes = append(instanceTypes, *it)
				}
			}
			nodeGroups = append(nodeGroups, models.NodeGroup{
				NodeGroupName:  aws.StringValue(ng.NodegroupName),
				NodeGroupARN:   aws.StringValue(ng.NodegroupArn),
				Status:         aws.StringValue(ng.Status),
				InstanceTypes:  instanceTypes,
				ScalingConfig:  scaling,
				ReleaseVersion: aws.StringValue(ng.ReleaseVersion),
				Labels:         labels,
				Tags:           ngTags,
			})
		}
		clusterModel.NodeGroups = nodeGroups
	}

	return clusterModel, nil
}

func (s *AWSService) ListEKSClustersByAccount(accountID string) ([]models.EKSCluster, error) {
	cacheKey := fmt.Sprintf("eks-clusters:%s", accountID)

	if cached, found := s.cache.Get(cacheKey); found {
		if clusters, ok := cached.([]models.EKSCluster); ok {
			return clusters, nil
		}
	}

	allClusters, err := s.ListEKSClusters()
	if err != nil {
		return nil, err
	}

	var accountClusters []models.EKSCluster
	for _, c := range allClusters {
		if c.AccountID == accountID {
			accountClusters = append(accountClusters, c)
		}
	}

	s.cache.Set(cacheKey, accountClusters, s.cacheTTL)

	return accountClusters, nil
}

func (s *AWSService) InvalidateEKSClustersCache() {
	s.cache.Delete("eks-clusters")
	s.cache.DeletePattern("eks-clusters:")
}

func (s *AWSService) getAccessibleRegions(sess *session.Session) ([]string, error) {
	ec2Client := ec2.New(sess)
	regionsResult, err := ec2Client.DescribeRegions(&ec2.DescribeRegionsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("opt-in-status"),
				Values: []*string{aws.String("opt-in-not-required"), aws.String("opted-in")},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	var regions []string
	for _, r := range regionsResult.Regions {
		if r.RegionName != nil {
			regions = append(regions, *r.RegionName)
		}
	}
	return regions, nil
}
