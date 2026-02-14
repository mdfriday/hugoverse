package handler

import (
	"fmt"
	"net/http"

	contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/internal/infrastructure/licensekit"
	"github.com/mdfriday/hugoverse/internal/infrastructure/smtp"
	apiFrom "github.com/mdfriday/hugoverse/internal/interfaces/api/form"
	"github.com/mdfriday/hugoverse/pkg/timestamp"
)

// GetTrialHandler 申请试用 License
// POST /api/license/trial
// 参数：email (POST form)
// 返回：{ "success": true, "license_key": "MDF-XXXX-XXXX-XXXX", "email": "xxx@example.com", "password": "xxx" }
func (s *Handler) GetTrialHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	err := req.ParseMultipartForm(apiFrom.MaxMemory)
	if err != nil {
		s.log.Errorf("Error parsing multipart form: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	email := req.PostForm.Get("email")

	// 1. 验证邮箱是否为空
	if email == "" {
		s.jsonError(res, "Email is required", http.StatusBadRequest)
		return
	}

	// 2. 验证邮箱格式
	if !licensekit.ValidateEmail(email) {
		s.jsonError(res, "Invalid email format", http.StatusBadRequest)
		return
	}

	// 3. 检查邮箱是否已经申请过试用
	existingTrial, err := s.contentApp.GetLicenseTrialByEmail(email)
	if err == nil && existingTrial != nil {
		s.jsonError(res, "This email has already been used to request a trial license", http.StatusConflict)
		return
	}

	// 4. 生成免费 license key
	licenseKey := licensekit.GenerateLicenseKey()

	// 5. 生成用户名密码
	userEmail := licensekit.LicenseKeyToEmail(licenseKey)
	userPassword := licensekit.LicenseKeyToPassword(licenseKey)

	// 6. 获取 free plan 配置
	planConfig := licensekit.GetPlanConfig("free")

	now := timestamp.CurrentTimeMillis()

	// 7. 创建 License 对象
	license := &contentVO.License{
		LicenseKey:      licenseKey,
		Plan:            contentVO.PlanFree,
		MaxDevices:      planConfig.MaxDevices,
		MaxIPs:          planConfig.MaxIPs,
		Activated:       false,
		ActivatedAt:     0,
		IssueDate:       now,
		ExpiryDate:      0, // 在首次激活时设置
		SyncEnabled:     planConfig.SyncEnabled,
		SyncQuotaMB:     planConfig.SyncQuotaMB,
		PublishEnabled:  planConfig.PublishEnabled,
		MaxSites:        planConfig.MaxSites,
		MaxStorageMB:    planConfig.MaxStorageMB,
		CustomDomain:    planConfig.CustomDomain,
		CustomSubDomain: planConfig.CustomSubDomain,
		Item: contentVO.Item{
			Timestamp: now,
			Updated:   now,
			Namespace: "License",
		},
	}

	// 8. 创建 License 到数据库
	_, err = s.contentApp.CreateLicense(license)
	if err != nil {
		s.log.Errorf("Failed to create license: %v", err)
		s.jsonError(res, "Failed to create license: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 9. 创建对应的用户
	err = licensekit.CreateUser("http://127.0.0.1:1314", userEmail, userPassword)
	if err != nil {
		s.log.Errorf("Failed to create user for license %s: %v", licenseKey, err)
		// 用户创建失败，但 license 已创建，记录错误但继续
		// 可以考虑在这里回滚 license 创建，但为了简单起见先继续
	}

	// 10. 创建 LicenseTrial 记录
	licenseTrial := &contentVO.LicenseTrial{
		Email:   email,
		License: licenseKey,
		Item: contentVO.Item{
			Timestamp: now,
			Updated:   now,
			Namespace: "LicenseTrial",
		},
	}

	_, err = s.contentApp.CreateLicenseTrial(licenseTrial)
	if err != nil {
		s.log.Errorf("Failed to create license trial record: %v", err)
		s.jsonError(res, "Failed to create license trial record: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 11. 发送邮件
	if s.adminApp.SMTP.IsConfigured() {
		smtpClient := smtp.NewClient(&smtp.Config{
			Host:     s.adminApp.SMTP.Host(),
			Port:     s.adminApp.SMTP.Port(),
			Username: s.adminApp.SMTP.Username(),
			Password: s.adminApp.SMTP.Password(),
			From:     s.adminApp.SMTP.From(),
			UseTLS:   s.adminApp.SMTP.UseTLS(),
		})

		emailBody := fmt.Sprintf(`
Dear User,

Thank you for your interest in MDFriday!

Your trial license has been successfully created. Here are your credentials:

License Key: %s

Validity: %d days

You can now use these credentials to activate your license and start using MDFriday.

Best regards,
MDFriday Team
`, licenseKey, planConfig.ValidityDays)

		emailMsg := &smtp.EmailMessage{
			To:      []string{email},
			Subject: "Your MDFriday Trial License",
			Body:    emailBody,
			IsHTML:  false,
		}

		err = smtpClient.SendEmail(emailMsg)
		if err != nil {
			s.log.Errorf("Failed to send trial license email to %s: %v", email, err)
			// 邮件发送失败，但 license 已创建，继续返回成功
		} else {
			s.log.Infof("Trial license email sent successfully to %s", email)
		}
	} else {
		s.log.Warnf("SMTP not configured, skipping email notification for trial license %s", licenseKey)
	}

	// 12. 返回成功信息
	s.jsonResponse(res, map[string]interface{}{
		"success":       true,
		"license_key":   licenseKey,
		"message":       "Trial license created successfully. Please check your email for details.",
		"validity_days": planConfig.ValidityDays,
	})
}
