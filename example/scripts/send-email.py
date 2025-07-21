#!/usr/bin/env python3

import smtplib
import os
import sys
import time
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart
from email.utils import formatdate

# Configuration
SMTP_HOST = os.environ.get('SMTP_HOST', 'localhost')
SMTP_PORT = int(os.environ.get('SMTP_PORT', '10025'))

def send_email(from_addr, to_addr, cc_addr=None, bcc_addr=None, subject="", body=""):
    """Send an email via SMTP"""
    # Create message
    msg = MIMEMultipart()
    msg['From'] = from_addr
    msg['To'] = to_addr
    if cc_addr:
        msg['Cc'] = cc_addr
    msg['Subject'] = subject
    msg['Date'] = formatdate(localtime=True)
    msg['Message-ID'] = f'<{int(time.time())}@fakesmtp.local>'
    
    # Add body
    msg.attach(MIMEText(body, 'plain'))
    
    # Build recipient list
    recipients = [to_addr]
    if cc_addr:
        recipients.append(cc_addr)
    if bcc_addr:
        recipients.append(bcc_addr)
    
    # Send email
    try:
        with smtplib.SMTP(SMTP_HOST, SMTP_PORT) as server:
            server.sendmail(from_addr, recipients, msg.as_string())
        print(f"✓ Email sent successfully: {subject}")
    except Exception as e:
        print(f"✗ Error sending email: {e}")
        return False
    
    return True

def main():
    print("=== Sending test emails to fake SMTP server ===")
    print(f"SMTP Server: {SMTP_HOST}:{SMTP_PORT}")
    print()
    
    # Test emails
    test_emails = [
        {
            "from": "sender@example.com",
            "to": "recipient@example.com",
            "subject": "Test Email 1",
            "body": "This is a simple test email."
        },
        {
            "from": "alice@example.com",
            "to": "bob@example.com",
            "cc": "charlie@example.com",
            "subject": "Meeting Invitation",
            "body": "Please join our meeting tomorrow at 10 AM."
        },
        {
            "from": "manager@company.com",
            "to": "team@company.com",
            "bcc": "hr@company.com",
            "subject": "Team Update",
            "body": "This is a confidential team update. HR is BCCed on this email."
        },
        {
            "from": "newsletter@example.com",
            "to": "user1@example.com",
            "cc": "user2@example.com",
            "bcc": "admin@example.com",
            "subject": "Newsletter #42",
            "body": "Welcome to our weekly newsletter!"
        },
        {
            "from": "support@example.com",
            "to": "customer@example.com",
            "subject": "Re: Question about your product",
            "body": "Thank you for your inquiry. We're happy to help!"
        }
    ]
    
    # Send all test emails
    for i, email in enumerate(test_emails, 1):
        print(f"{i}. Sending: {email['subject']}")
        send_email(
            from_addr=email["from"],
            to_addr=email["to"],
            cc_addr=email.get("cc"),
            bcc_addr=email.get("bcc"),
            subject=email["subject"],
            body=email["body"]
        )
        # Small delay between emails
        time.sleep(0.5)
    
    print()
    print("=== All emails sent! ===")
    print(f"Check the emails at: http://{SMTP_HOST}:11080/")

if __name__ == "__main__":
    main()