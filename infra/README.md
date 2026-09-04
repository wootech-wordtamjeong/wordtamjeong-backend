# Wordtamjeong AWS infrastructure

이 디렉터리는 다음 역할로 분리되어 있습니다.

- Terraform: VPC, 네트워크, ECR, IAM, EC2, Elastic IP 생성
- Ansible: EC2에 Docker를 설치하고 애플리케이션 배포
- GitHub Actions: 테스트, ECR 이미지 푸시, Ansible 배포 실행

기존 운영 리소스와 충돌하지 않도록 먼저 `environment = "dev"`로 새 환경을
만드는 것을 권장합니다. 기존 ECR처럼 이름이 같은 리소스를 유지하려면 새로
생성하지 말고 Terraform import로 state에 편입해야 합니다.

## 1. 사전 준비

다음 항목이 필요합니다.

- AWS CLI 인증
- Terraform 1.6 이상
- Ansible을 실행할 Linux, WSL 또는 GitHub Actions 환경
- AWS EC2 Key Pair
- 현재 관리 PC의 공인 IP(`/32`)
- 사용할 Bedrock 모델에 대한 Model access

Terraform과 Ansible은 Windows 네이티브 환경보다 WSL2에서 실행하는 것이
간단합니다.

## 2. Terraform 변수 설정

```bash
cd infra/terraform
cp terraform.tfvars.example terraform.tfvars
```

`terraform.tfvars`에서 최소한 다음 값을 실제 환경에 맞게 변경합니다.

```hcl
admin_cidr  = "YOUR_PUBLIC_IP/32"
ssh_key_name = "YOUR_EXISTING_EC2_KEY_PAIR"

bedrock_model_arns = [
  "arn:aws:bedrock:ap-northeast-2::foundation-model/amazon.titan-embed-text-v2:0",
]
```

Terraform state에는 비밀값을 넣지 않습니다. `ADMIN_API_KEY`는 Terraform 변수가
아니며 GitHub Actions secret으로 관리합니다.

초기 로컬 state로 검증할 때는 `backend.tf.example`을 그대로 두십시오. 원격
state로 전환할 때 별도의 state용 S3 버킷을 먼저 만든 후 `backend.tf`로 복사하고
값을 수정합니다.

```bash
terraform init
terraform fmt -check -recursive
terraform validate
terraform plan -out wordtamjeong.tfplan
terraform apply wordtamjeong.tfplan
terraform output
```

중요 출력값은 다음과 같습니다.

- `server_public_ip`: Ansible 및 GitHub의 EC2 주소
- `ecr_repository_url`: 배포 이미지 주소
- `github_deploy_role_arn`: GitHub OIDC 역할 ARN
- `security_group_id`: GitHub 러너의 임시 SSH 접근에 사용하는 보안 그룹

## 3. GitHub OIDC 설정

AWS 계정에 `token.actions.githubusercontent.com` OIDC provider가 이미 있다면 그
ARN을 `github_oidc_provider_arn`에 설정하십시오. Terraform이 master 브랜치만
역할을 맡을 수 있는 ECR push 역할을 생성합니다.

OIDC provider는 AWS 계정당 공유하는 리소스라 이 프로젝트에서 무조건 생성하지
않습니다. 없는 계정에서는 IAM의 Identity providers에서 다음 값으로 한 번만
생성합니다.

```text
Provider URL: https://token.actions.githubusercontent.com
Audience: sts.amazonaws.com
```

그 후 다시 `terraform apply`하고 `github_deploy_role_arn` 출력을 확인합니다.

## 4. 최초 EC2 구성

Terraform 출력 IP로 inventory를 만듭니다.

```bash
cd infra/ansible
cp inventory/hosts.ini.example inventory/hosts.ini
export ANSIBLE_CONFIG="$PWD/ansible.cfg"
```

`inventory/hosts.ini`의 `ansible_host`를 `server_public_ip`로 변경한 다음 실행합니다.

```bash
ansible-playbook bootstrap.yml --private-key /path/to/key.pem
```

WSL의 `/mnt/c` 아래 저장소는 Windows 권한 매핑 때문에 Ansible이 현재 디렉터리의
`ansible.cfg`를 자동으로 무시할 수 있습니다. 위와 같이 `ANSIBLE_CONFIG`를
명시하면 저장소의 inventory와 설정이 정상 적용됩니다.

이 단계는 Docker, Docker Compose, AWS CLI와 기본 패키지만 설치합니다. 반복해서
실행해도 같은 서버 상태를 유지합니다.

## 5. 수동 배포 확인

GitHub Actions 연결 전에 한 번 수동 배포할 수 있습니다.

```bash
export IMAGE_URI="ACCOUNT_ID.dkr.ecr.ap-northeast-2.amazonaws.com/wordtamjeong:latest"
export AWS_REGION="ap-northeast-2"
export AWS_BEDROCK_MODEL="amazon.titan-embed-text-v2:0"
export ADMIN_API_KEY="REPLACE_WITH_A_LONG_RANDOM_VALUE"
export ALLOWED_ORIGINS="https://your-frontend.example.com"
export ENABLE_INTERNAL_SCHEDULER="true"

ansible-playbook deploy.yml --private-key /path/to/key.pem
```

EC2는 Access Key 없이 Instance Profile로 ECR과 Bedrock에 접근합니다. 배포 시
`data/words.txt`를 복사하고 `/opt/kkomantl/data`를 컨테이너의 `/app/data`에
연결하므로 컨테이너를 교체해도 생성된 퀴즈가 남습니다.

새 날짜의 퀴즈 캐시가 없으면 937개 단어의 임베딩을 먼저 계산하므로 최초
health check가 수 분 걸릴 수 있습니다.

## 6. GitHub 설정

Repository settings에서 다음 값을 등록합니다.

Actions secrets:

```text
AWS_ROLE_ARN       terraform output github_deploy_role_arn
EC2_HOST           terraform output server_public_ip
EC2_SSH_KEY        EC2 Key Pair의 private key 전체
ADMIN_API_KEY      충분히 긴 임의 문자열
```

Actions variables:

```text
AWS_REGION                ap-northeast-2
ECR_REPOSITORY            wordtamjeong
AWS_BEDROCK_MODEL          amazon.titan-embed-text-v2:0
ALLOWED_ORIGINS            실제 프런트엔드 origin
ENABLE_INTERNAL_SCHEDULER  true
EC2_SECURITY_GROUP_ID       terraform output security_group_id
```

워크플로는 PR과 develop push에서는 테스트만 실행합니다. master push에 한해서
OIDC로 AWS 역할을 맡고, 커밋 SHA 이미지를 ECR에 올린 후 Ansible로 배포합니다.
배포 직전에는 해당 GitHub 러너의 IP에만 SSH를 허용하고, 배포 성공 여부와 관계없이
마지막 단계에서 그 규칙을 제거합니다.

## 7. 스케줄러 선택

현재 서버에는 KST 자정 내부 스케줄러가 있습니다. `ENABLE_INTERNAL_SCHEDULER`의
기본값은 `true`입니다.

나중에 EventBridge와 Lambda를 Terraform으로 옮길 때는 이 값을 `false`로 바꿔
두 스케줄러가 동시에 `/admin/rotate`를 실행하지 않게 하십시오. EC2 배포가 먼저
안정화된 후 Lambda/EventBridge를 추가하는 것이 안전합니다.

## 운영 전 후속 개선

- SSH 대신 SSM Session Manager 기반 Ansible 연결
- ALB, ACM 인증서 및 HTTPS 적용
- 8080 포트를 ALB Security Group에만 허용
- 관리자 키를 SSM Parameter Store 또는 Secrets Manager로 이전
- 애플리케이션 시작과 대규모 임베딩 사전 계산 분리
- 별도 S3 state 버킷과 S3 lockfile 활성화
