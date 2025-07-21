# go-fake-smtp-server Example

このディレクトリには、go-fake-smtp-serverの外部SMTP連携テストの実装例が含まれています。

## 構成

```
example/
├── docker/
│   ├── Dockerfile          # アプリケーションのDockerイメージ
│   └── docker-compose.yml  # Docker Compose設定
├── scripts/
│   ├── send-email.py       # SMTP経由でメールを送信するPythonスクリプト
│   └── test-search-api.sh  # 検索APIをテストするスクリプト
└── README.md
```

## 使用方法

### 1. Dockerでアプリケーションを起動

```bash
cd example/docker
docker-compose up -d
```

これにより以下のポートでサービスが起動します：
- SMTP: `localhost:10025`
- HTTP API: `localhost:11080`

### 2. テストメールを送信

```bash
cd example/scripts
./send-email.py
# または
python3 send-email.py
```

このスクリプトは以下のテストメールを送信します：
1. シンプルなメール
2. CC付きメール
3. BCC付きメール
4. 複数の宛先を持つメール
5. 特殊文字を含むメール

### 3. 検索APIをテスト

```bash
cd example/scripts
./test-search-api.sh
```

このスクリプトは以下のAPIエンドポイントをテストします：
- `/` - 全メール取得
- `/search/to` - To検索
- `/search/cc` - CC検索
- `/search/bcc` - BCC検索
- `/search/from` - From検索

### 4. 手動でメールを確認

ブラウザで以下のURLにアクセス：
```
http://localhost:11080/
```

または、curlで確認：
```bash
curl http://localhost:11080/ | jq
```

## 環境変数

スクリプトは以下の環境変数でカスタマイズ可能です：

```bash
# SMTPホストとポート
export SMTP_HOST=localhost
export SMTP_PORT=10025

# APIホストとポート
export API_HOST=localhost
export API_PORT=11080
```

## クリーンアップ

```bash
cd example/docker
docker-compose down
```

## 依存関係

- Docker & Docker Compose
- Python 3 - メール送信用
- curl - API通信用
- jq (オプション) - JSON整形用

### ポートが使用中の場合

docker-compose.ymlでポートマッピングを変更してください：
```yaml
ports:
  - "20025:10025"  # 左側の数字を変更
  - "21080:11080"  # 左側の数字を変更
```