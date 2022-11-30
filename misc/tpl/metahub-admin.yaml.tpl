minio:
  url: {{ MINIO_URL }}
  region: {{ MINIO_REGION |default("us-east-1") }}
  user: {{ MINIO_USER |default("minio")}}
  password: {{ MINIO_PASSWD|default("miniominio") }}
  redirect: {{ MINIO_REDIRECT|default("false") }}
  ssl: {{ MINIO_SSL|default("false") }}
  manifests:
    bucket: {{ MINIO_BUCKET_MANIFESTS|default("manifests") }}
  configs:
    bucket: {{ MINIO_BUCKET_CONFIGS|default("configs") }}
  layers:
    bucket: {{ MINIO_BUCKET_LAYERS|default("layers") }}

neo4j:
  dbname: neo4j
  user: {{ NEO4J_USER|default("neo4j") }}
  password: {{ NEO4J_PASSWD|default("test") }}
  url: {{ NEO4J_URL }}