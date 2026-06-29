CREATE ROLE auth_app LOGIN PASSWORD 'auth_app_dev';
CREATE DATABASE auth_system OWNER auth_app;

CREATE ROLE file_app LOGIN PASSWORD 'file_app_dev';
CREATE DATABASE file_system OWNER file_app;

CREATE ROLE knowledge_app LOGIN PASSWORD 'knowledge_app_dev';
CREATE DATABASE knowledge_system OWNER knowledge_app;

CREATE ROLE qa_app LOGIN PASSWORD 'qa_app_dev';
CREATE DATABASE qa_system OWNER qa_app;

CREATE ROLE document_app LOGIN PASSWORD 'document_app_dev';
CREATE DATABASE document_system OWNER document_app;

CREATE ROLE ai_gateway_app LOGIN PASSWORD 'ai_gateway_app_dev';
CREATE DATABASE ai_gateway_system OWNER ai_gateway_app;

\connect qa_system
CREATE EXTENSION IF NOT EXISTS pgcrypto;

\connect document_system
CREATE EXTENSION IF NOT EXISTS pgcrypto;
