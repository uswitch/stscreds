# stscreds

stscreds makes it easier to work with temporary AWS API keys and, by extension, easier to stop using long-term credentials.

## Rationale

Working with Amazon libraries often requires a developer to use an Access Key and Secret Key pair. Once created these don't expire (until you remove them).

Amazon's [Security Token Service](http://docs.aws.amazon.com/STS/latest/APIReference/Welcome.html) can be used to request temporary credentials that automatically expire and must be requested/authenticated with an [MFA device](https://aws.amazon.com/iam/details/mfa/).

Further, through applying a policy to users, it's possible to restrict API access so that privileged operations are only allowed when the credentials were authenticated using an MFA device.

This tool helps make it easier to work with temporary credentials and shows a sample policy for restricting access to privileged APIs without MFA authentication.

## Usage
Once installed you can use the tool as follows:

```
$ stscreds auth
Current user: john.doe. Please enter MFA token: XXXXXX
Wrote credentials to /home/foo/.aws/credentials
```

## Installing

You can download binary releases (for Linux and Darwin) from GitHub: [https://github.com/uswitch/stscreds/releases](https://github.com/uswitch/stscreds/releases). Alternatively, you can also build from source using [Go](https://golang.org):

```
$ go get github.com/uswitch/stscreds
```

## Setup
### IAM Policy
Although stscreds can be used just to create temporary credentials, it's better to restrict API access to ensure only a handful of APIs are usable without using the credentials stscreds provides.

The following policy provides an example, allowing `sts:GetSessionToken`, `iam:GetUser` and `iam:ListMFADevices` (the 3 API methods stscreds uses to setup/authenticate) to users when authenticating using regular long-term credentials (such as those retrieved from the AWS Console). All other API operations require credentials generated with `sts:GetSessionToken`.

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": ...,
            "Effect": "Allow",
            "Action": [
                "sts:GetSessionToken",
                "iam:GetUser",
                "iam:ListMFADevices"
            ],
            "Resource": [
                "arn:aws:iam::*:user/${aws:username}"
            ]
        },
        {
            "Sid": ...,
            "Effect": "Allow",
            "Action": [
                "*"
            ],
            "Condition": {
                "Bool": {
                    "aws:MultiFactorAuthPresent": "true"
                }
            },
            "Resource": [
                "*"
            ]
        }
    ]
}
```

The above policy is just an example. It's a good idea to ensure your policies control access to privileged and/or destructive APIs. In the policy above the key part is to ensure you add a condition on `aws:MultiFactorAuthPresent`.

### Configuring the tool

stscreds uses `~/.stscreds/credentials` to store long-term API keys (these will often be the ones currently in use) and are the same keys generated/downloaded from the AWS Console.

```
$ stscreds init
AWS Access Key: XXXXXXX
AWS Secret Access Key: XXXXXXX
Successfully wrote /home/foo/.stscreds/credentials
```

## Generating new keys

Once you've initialised using `stscreds init` above you'll only need to run `stscreds auth` from thereon. 

```
$ stscreds auth
Current user: first.last. Please enter MFA token: XXXXXX
Wrote credentials to /home/foo/.aws/credentials

$ cat ~/.aws/credentials
[default]
aws_access_key_id     = FOO
aws_secret_access_key = BAR
aws_session_token     = BAZ
```

## Reading credentials/Setting env variables

If you want to set environment variables from the stored `~/.aws/credentials` (having run `stscreds auth`) you can use the `read` command. For example, inside your `~/.bashrc` you could use:

```
export AWS_ACCESS_KEY_ID=$(stscreds read aws_access_key_id)
export AWS_SECRET_ACCESS_KEY=$(stscreds read aws_secret_access_key)
export AWS_SESSION_TOKEN=$(stscreds read aws_session_token)
```

`read` will also ensure credentials are up-to-date; if credentials need to be refreshed you'll be prompted to enter another MFA token.
