# SMTP Configuration Guide for Authentication

## Overview

The authentication system requires SMTP credentials to send:
- **Email verification** links when users sign up
- **Password reset** links when users forget their password

Currently, the SMTP password is set to a placeholder value `CONFIGURE_SMTP_PASSWORD` and **must be configured** for email functionality to work.

---

## Option 1: SendGrid (Recommended for Production)

SendGrid is a reliable email service with a free tier (100 emails/day) and excellent deliverability.

### Steps:

1. **Create SendGrid Account**
   - Go to https://sendgrid.com/
   - Sign up for a free account
   - Verify your email address

2. **Create API Key**
   - Go to Settings → API Keys
   - Click "Create API Key"
   - Name it "InvestorCenter Auth"
   - Set permissions to "Full Access" (or "Mail Send" only)
   - Copy the API key (you'll only see it once!)

3. **Update Kubernetes Secret**
   ```bash
   # Replace YOUR_SENDGRID_API_KEY with the actual key from step 2
   kubectl patch secret app-secrets -n investorcenter --type=json -p='[
     {"op": "replace", "path": "/data/smtp-host", "value": "'$(echo -n 'smtp.sendgrid.net' | base64)'"},
     {"op": "replace", "path": "/data/smtp-username", "value": "'$(echo -n 'apikey' | base64)'"},
     {"op": "replace", "path": "/data/smtp-password", "value": "'$(echo -n 'YOUR_SENDGRID_API_KEY' | base64)'"}
   ]'
   ```

4. **Restart Backend Pods**
   ```bash
   kubectl rollout restart deployment/investorcenter-backend -n investorcenter
   kubectl rollout status deployment/investorcenter-backend -n investorcenter
   ```

5. **Verify Configuration**
   ```bash
   # Check logs for email sending
   kubectl logs -n investorcenter -l app=investorcenter-backend --tail=50 | grep -i "email\|smtp"
   ```

### SendGrid Configuration Summary:
- **SMTP Host:** `smtp.sendgrid.net`
- **SMTP Port:** `587`
- **SMTP Username:** `apikey` (literally the word "apikey")
- **SMTP Password:** Your SendGrid API key

---

## Option 2: Gmail (Good for Testing)

Gmail can be used for testing but has strict sending limits (500 emails/day).

### Steps:

1. **Enable 2-Factor Authentication**
   - Go to https://myaccount.google.com/security
   - Enable 2-Step Verification if not already enabled

2. **Generate App Password**
   - Go to https://myaccount.google.com/apppasswords
   - Select app: "Mail"
   - Select device: "Other (Custom name)" → Enter "InvestorCenter"
   - Click "Generate"
   - Copy the 16-character password (remove spaces)

3. **Update Kubernetes Secret**
   ```bash
   # Replace with your Gmail address and app password
   kubectl patch secret app-secrets -n investorcenter --type=json -p='[
     {"op": "replace", "path": "/data/smtp-host", "value": "'$(echo -n 'smtp.gmail.com' | base64)'"},
     {"op": "replace", "path": "/data/smtp-username", "value": "'$(echo -n 'your-email@gmail.com' | base64)'"},
     {"op": "replace", "path": "/data/smtp-password", "value": "'$(echo -n 'YOUR_16_CHAR_APP_PASSWORD' | base64)'"}
   ]'
   ```

4. **Update "From" Email (Optional)**

   By default, emails are sent from `noreply@investorcenter.ai`. If using Gmail, you may want to use your Gmail address:

   ```bash
   kubectl edit deployment investorcenter-backend -n investorcenter
   ```

   Find the `SMTP_FROM_EMAIL` environment variable and change it to your Gmail address:
   ```yaml
   - name: SMTP_FROM_EMAIL
     value: "your-email@gmail.com"
   ```
   Save and exit (the deployment will automatically roll out).

5. **Restart Backend Pods**
   ```bash
   kubectl rollout restart deployment/investorcenter-backend -n investorcenter
   ```

### Gmail Configuration Summary:
- **SMTP Host:** `smtp.gmail.com`
- **SMTP Port:** `587`
- **SMTP Username:** Your Gmail address (e.g., `your-email@gmail.com`)
- **SMTP Password:** 16-character app password (no spaces)

---

## Option 3: AWS SES (For AWS-Heavy Setups)

If you're already using AWS, Simple Email Service (SES) is a cost-effective option.

### Steps:

1. **Verify Domain or Email**
   - Go to AWS SES Console
   - Verify your domain (investorcenter.ai) OR verify a specific email address
   - Move out of sandbox mode (requires support ticket)

2. **Create SMTP Credentials**
   - In SES Console → SMTP Settings
   - Click "Create My SMTP Credentials"
   - Download the credentials

3. **Update Kubernetes Secret**
   ```bash
   kubectl patch secret app-secrets -n investorcenter --type=json -p='[
     {"op": "replace", "path": "/data/smtp-host", "value": "'$(echo -n 'email-smtp.us-east-1.amazonaws.com' | base64)'"},
     {"op": "replace", "path": "/data/smtp-username", "value": "'$(echo -n 'YOUR_SES_SMTP_USERNAME' | base64)'"},
     {"op": "replace", "path": "/data/smtp-password", "value": "'$(echo -n 'YOUR_SES_SMTP_PASSWORD' | base64)'"}
   ]'
   ```

4. **Restart Backend Pods**
   ```bash
   kubectl rollout restart deployment/investorcenter-backend -n investorcenter
   ```

### AWS SES Configuration Summary:
- **SMTP Host:** `email-smtp.us-east-1.amazonaws.com` (or your region)
- **SMTP Port:** `587`
- **SMTP Username:** From SES SMTP credentials
- **SMTP Password:** From SES SMTP credentials

---

## Testing Email Configuration

Once SMTP is configured, test it by signing up a new user:

### 1. Test Signup (Should Send Verification Email)

```bash
curl -X POST https://api.investorcenter.ai/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "testpassword123",
    "full_name": "Test User"
  }'
```

**Expected Response:**
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "expires_in": 3600,
  "user": {
    "id": "...",
    "email": "test@example.com",
    "full_name": "Test User",
    "email_verified": false
  }
}
```

### 2. Check for Verification Email

- Check the inbox of `test@example.com`
- Look in spam folder if not found
- Email should have subject: "Verify your InvestorCenter.ai account"
- Email should contain a verification link

### 3. Check Backend Logs

```bash
# Look for email sending logs
kubectl logs -n investorcenter -l app=investorcenter-backend --tail=100 | grep -i "email\|smtp"
```

**Success logs should show:**
```
Sending verification email to test@example.com
Email sent successfully
```

**Error logs might show:**
```
Failed to send email: SMTP authentication failed
Failed to send email: connection refused
```

### 4. Test Password Reset

```bash
curl -X POST https://api.investorcenter.ai/api/v1/auth/forgot-password \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com"}'
```

Check email for password reset link.

---

## Troubleshooting

### Problem: "SMTP authentication failed"

**Solution:**
- Double-check your SMTP username and password
- For SendGrid: Ensure username is literally "apikey"
- For Gmail: Ensure you're using an app password, not your regular password
- Verify the secret was updated correctly:
  ```bash
  kubectl get secret app-secrets -n investorcenter -o jsonpath='{.data.smtp-password}' | base64 -d
  ```

### Problem: Emails not arriving

**Possible causes:**
1. **Check spam folder** - Emails from new senders often go to spam
2. **Domain reputation** - New domains may have deliverability issues
3. **Rate limiting** - You may have hit sending limits (Gmail: 500/day)
4. **Verification required** - Some services require domain verification

**Debug steps:**
```bash
# Check if email was sent from backend
kubectl logs -n investorcenter -l app=investorcenter-backend --tail=200 | grep -i "email"

# Check SMTP connection
kubectl exec -n investorcenter -it investorcenter-backend-XXXXX-XXXXX -- sh
# Inside pod:
nc -zv smtp.sendgrid.net 587  # Should connect
wget --spider smtp.gmail.com:587  # Should connect
```

### Problem: "Connection refused" or "Connection timeout"

**Solution:**
- Check that SMTP port 587 is not blocked by firewall
- Verify SMTP host is correct
- Ensure pods have internet access

### Problem: Emails sent but verification link doesn't work

**Check:**
1. **FRONTEND_URL** environment variable is set correctly
2. Token hasn't expired (24 hours for email verification)
3. Database has the verification token stored

```bash
# Check FRONTEND_URL
kubectl get deployment investorcenter-backend -n investorcenter -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="FRONTEND_URL")].value}'

# Should output: https://investorcenter.ai
```

---

## Security Best Practices

1. **Never commit SMTP credentials** to Git
   - Credentials are stored in Kubernetes secrets
   - Secrets are excluded from version control

2. **Use app-specific passwords**
   - Don't use your main account password
   - Use API keys (SendGrid) or app passwords (Gmail)

3. **Rotate credentials periodically**
   - Generate new API keys every 90 days
   - Update Kubernetes secret

4. **Monitor email sending**
   - Set up alerts for high email volume (possible spam)
   - Monitor bounce rates

5. **Use a dedicated sending domain**
   - Configure SPF, DKIM, and DMARC records
   - Improves deliverability and reduces spam classification

---

## Production Checklist

Before going live:

- [ ] SMTP credentials configured (not placeholder)
- [ ] Test email verification flow end-to-end
- [ ] Test password reset flow end-to-end
- [ ] Emails arrive in inbox (not spam)
- [ ] Email links work correctly (FRONTEND_URL is correct)
- [ ] Backend logs show successful email sending
- [ ] Set up monitoring for email failures
- [ ] Configure email rate limiting (to prevent abuse)
- [ ] Add domain to SendGrid/SES sender verification
- [ ] Configure SPF/DKIM/DMARC for better deliverability

---

## Quick Reference Commands

**Check current SMTP configuration:**
```bash
kubectl get secret app-secrets -n investorcenter -o jsonpath='{.data}' | jq 'to_entries | map({key, value: (.value | @base64d)}) | from_entries'
```

**Update SMTP password only:**
```bash
kubectl patch secret app-secrets -n investorcenter --type=json -p='[
  {"op": "replace", "path": "/data/smtp-password", "value": "'$(echo -n 'NEW_PASSWORD' | base64)'"}
]'
kubectl rollout restart deployment/investorcenter-backend -n investorcenter
```

**Check backend logs for email issues:**
```bash
kubectl logs -n investorcenter -l app=investorcenter-backend --tail=100 -f | grep -i "email\|smtp"
```

**Test SMTP connection from pod:**
```bash
kubectl exec -n investorcenter -it $(kubectl get pods -n investorcenter -l app=investorcenter-backend -o jsonpath='{.items[0].metadata.name}') -- sh
# Inside pod:
nc -zv smtp.sendgrid.net 587
```

---

## Next Steps

After configuring SMTP:

1. ✅ Verify emails are being sent
2. ✅ Test full user signup flow
3. ✅ Test password reset flow
4. ✅ Monitor email delivery rates
5. Configure SendGrid webhooks for bounce tracking (optional)
6. Set up custom email templates with branding (optional)

---

**Need Help?**

- **SendGrid Docs:** https://docs.sendgrid.com/
- **Gmail App Passwords:** https://support.google.com/accounts/answer/185833
- **AWS SES:** https://docs.aws.amazon.com/ses/

**Common Issues:** See [Production Deployment Guide](./phase1-production-deployment-guide.md)
