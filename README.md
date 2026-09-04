# 워드탐정 백엔드

한국어 의미 유사도 기반 단어 추측 게임의 백엔드 서버

## 기술 스택

- Go 1.21+
- AWS SDK for Go v2
- AWS Bedrock (Titan/Cohere Embeddings)
- net/http (표준 라이브러리)

## 프로젝트 구조

```
backend/
├── cmd/
│   ├── server/          # 메인 HTTP 서버
│   └── lambda/          # Lambda 함수 (일일 정답 자동 갱신)
├── internal/
│   ├── handlers/        # HTTP 핸들러
│   ├── models/          # 데이터 모델
│   └── services/        # 비즈니스 로직
├── pkg/
│   ├── bedrock/         # AWS Bedrock 연동
│   └── utils/           # 유틸리티 (코사인 유사도 등)
└── data/                # 퀴즈 데이터 저장소
```

## 설정

### 환경 변수

필수 환경 변수:
- `AWS_REGION`: AWS 리전 (기본값: ap-northeast-2)
- `AWS_BEDROCK_MODEL`: Bedrock 모델 ID
- `PORT`: 서버 포트 (기본값: 8080)
- `ADMIN_API_KEY`: 관리자 API 키

### 단어 목록

`data/words.txt` 파일에 한 줄에 하나씩 한국어 단어를 추가하세요:

```
사랑
행복
희망
...
```

## 실행

### 로컬 개발

```bash
# 의존성 설치
go mod download

# 서버 실행
go run cmd/server/main.go
```

### 빌드

```bash
# 서버 빌드
go build -o kkomantl-server ./cmd/server

# Lambda 빌드 (Linux용)
cd cmd/lambda
GOOS=linux GOARCH=amd64 go build -o bootstrap main.go
zip function.zip bootstrap
```

## AWS 인프라 및 배포

Terraform, Ansible, GitHub Actions를 이용한 AWS 배포 파일은 `infra/`에 있습니다.
새 환경을 만드는 순서와 필요한 AWS/GitHub 설정은
[`infra/README.md`](infra/README.md)를 참고하세요.

## API 엔드포인트

### GET /health

서버 상태 확인

**응답**
```json
{
  "status": "ok"
}
```

### GET /api/status

현재 퀴즈 정보 조회

**응답**
```json
{
  "quizId": 1322,
  "date": "2025-11-13"
}
```

### POST /api/guess

단어 추측 제출

**요청**
```json
{
  "word": "사랑"
}
```

**응답**
```json
{
  "word": "사랑",
  "similarity": 0.85,
  "rank": 5
}
```

- `similarity`: 0~1 사이의 유사도 값
- `rank`: 유사도 순위 (top 1000 이내인 경우), 없으면 null

### POST /admin/rotate

새로운 퀴즈 생성 (관리자 전용)

**헤더**
```
X-API-Key: your-admin-api-key
```

**요청**
```json
{
  "answer": "행복"
}
```

`answer`가 비어있으면 무작위 단어가 선택됩니다.

**응답**
```json
{
  "quizId": 1323,
  "answer": "행복",
  "date": "2025-11-14"
}
```

## AWS Bedrock 모델

### Titan Text Embedding v2
- 모델 ID: `amazon.titan-embed-text-v2:0`
- 한국어 지원
- 벡터 차원: 1024 (기본)

### Cohere Embed v4
- 모델 ID: `cohere.embed-multilingual-v4`
- 100개 이상 언어 지원
- 벡터 차원: 1024

## 주의사항

1. **AWS 자격 증명**: 로컬에서는 AWS CLI 설정 또는 환경 변수 필요, EC2에서는 IAM 역할 사용
2. **API 키**: `ADMIN_API_KEY`는 반드시 안전하게 관리
3. **단어 목록**: 더 많은 단어를 추가할수록 게임이 풍부해집니다
4. **캐싱**: top 1000 단어는 퀴즈 생성 시 미리 계산되어 `data/` 디렉토리에 저장됩니다

## 라이선스

MIT
