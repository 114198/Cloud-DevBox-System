-- Cloud DevBox Template Seeds
-- Seeds: 001_templates
-- Created: 2026-01-09
-- Updated: 2026-01-09 - Added 30+ preconfigured templates

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Insert system user for default templates
INSERT INTO users (id, email, username, display_name, role, quotas)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'system@devbox.io',
    'system',
    'System',
    'admin',
    '{"maxEnvironments": 1000, "maxCPU": 1000, "maxMemory": 1000, "maxStorage": 10000, "maxRunningTime": 100000}'
) ON CONFLICT (id) DO NOTHING;

-- ============================================
-- FRONTEND TEMPLATES (前端)
-- ============================================

-- React 18
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('react-18', 'React 18', 'React 18 + TypeScript + Vite 现代前端开发环境', '前端', 
 ARRAY['React', 'TypeScript', 'Vite', 'SPA'], '⚛️',
 '{"language": "javascript", "version": "18", "framework": "react", "buildTool": "vite"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "10Gi", "ports": [{"container": 5173, "protocol": "http"}]}',
 'FROM node:18-alpine
WORKDIR /app
RUN npm install -g pnpm
EXPOSE 5173
CMD ["pnpm", "dev", "--host"]',
 'pnpm create vite . --template react-ts && pnpm install',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Vue 3
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('vue-3', 'Vue 3', 'Vue 3 + TypeScript + Vite 响应式前端开发环境', '前端',
 ARRAY['Vue', 'TypeScript', 'Vite', 'Composition API'], '💚',
 '{"language": "javascript", "version": "18", "framework": "vue", "buildTool": "vite"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "10Gi", "ports": [{"container": 5173, "protocol": "http"}]}',
 'FROM node:18-alpine
WORKDIR /app
RUN npm install -g pnpm
EXPOSE 5173
CMD ["pnpm", "dev", "--host"]',
 'pnpm create vite . --template vue-ts && pnpm install',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Angular 17
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('angular-17', 'Angular 17', 'Angular 17 + TypeScript 企业级前端框架', '前端',
 ARRAY['Angular', 'TypeScript', 'RxJS', 'Enterprise'], '🅰️',
 '{"language": "typescript", "version": "18", "framework": "angular", "buildTool": "angular-cli"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "15Gi", "ports": [{"container": 4200, "protocol": "http"}]}',
 'FROM node:18-alpine
WORKDIR /app
RUN npm install -g @angular/cli pnpm
EXPOSE 4200
CMD ["ng", "serve", "--host", "0.0.0.0"]',
 'ng new app --skip-git --style=scss --routing=true && mv app/* . && rm -rf app',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Svelte
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('svelte', 'Svelte', 'Svelte + TypeScript + Vite 编译时前端框架', '前端',
 ARRAY['Svelte', 'TypeScript', 'Vite', 'Compiler'], '🔥',
 '{"language": "javascript", "version": "18", "framework": "svelte", "buildTool": "vite"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "10Gi", "ports": [{"container": 5173, "protocol": "http"}]}',
 'FROM node:18-alpine
WORKDIR /app
RUN npm install -g pnpm
EXPOSE 5173
CMD ["pnpm", "dev", "--host"]',
 'pnpm create vite . --template svelte-ts && pnpm install',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Solid.js
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('solidjs', 'Solid.js', 'Solid.js + TypeScript 高性能响应式框架', '前端',
 ARRAY['Solid.js', 'TypeScript', 'Vite', 'Reactive'], '💎',
 '{"language": "javascript", "version": "18", "framework": "solidjs", "buildTool": "vite"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "10Gi", "ports": [{"container": 3000, "protocol": "http"}]}',
 'FROM node:18-alpine
WORKDIR /app
RUN npm install -g pnpm
EXPOSE 3000
CMD ["pnpm", "dev", "--host"]',
 'npx degit solidjs/templates/ts . && pnpm install',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- ============================================
-- FULLSTACK TEMPLATES (全栈)
-- ============================================

-- Next.js 14
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('nextjs-14', 'Next.js 14', 'Next.js 14 + React + TypeScript 全栈 React 框架', '全栈',
 ARRAY['Next.js', 'React', 'TypeScript', 'SSR', 'App Router'], '▲',
 '{"language": "javascript", "version": "18", "framework": "nextjs", "buildTool": "next"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "15Gi", "ports": [{"container": 3000, "protocol": "http"}]}',
 'FROM node:18-alpine
WORKDIR /app
RUN npm install -g pnpm
EXPOSE 3000
CMD ["pnpm", "dev"]',
 'pnpm create next-app . --typescript --tailwind --eslint --app --src-dir --import-alias "@/*"',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Nuxt 3
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('nuxt-3', 'Nuxt 3', 'Nuxt 3 + Vue 3 + TypeScript 全栈 Vue 框架', '全栈',
 ARRAY['Nuxt', 'Vue', 'TypeScript', 'SSR', 'Nitro'], '💚',
 '{"language": "javascript", "version": "18", "framework": "nuxt", "buildTool": "nuxt"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "15Gi", "ports": [{"container": 3000, "protocol": "http"}]}',
 'FROM node:18-alpine
WORKDIR /app
RUN npm install -g pnpm
EXPOSE 3000
CMD ["pnpm", "dev", "--host"]',
 'pnpm dlx nuxi@latest init . && pnpm install',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- SvelteKit
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('sveltekit', 'SvelteKit', 'SvelteKit + TypeScript 全栈 Svelte 框架', '全栈',
 ARRAY['SvelteKit', 'Svelte', 'TypeScript', 'SSR', 'Vite'], '🔥',
 '{"language": "javascript", "version": "18", "framework": "sveltekit", "buildTool": "vite"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "15Gi", "ports": [{"container": 5173, "protocol": "http"}]}',
 'FROM node:18-alpine
WORKDIR /app
RUN npm install -g pnpm
EXPOSE 5173
CMD ["pnpm", "dev", "--host"]',
 'pnpm create svelte@latest . && pnpm install',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Remix
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('remix', 'Remix', 'Remix + React + TypeScript 全栈 Web 框架', '全栈',
 ARRAY['Remix', 'React', 'TypeScript', 'SSR', 'Web Standards'], '💿',
 '{"language": "javascript", "version": "18", "framework": "remix", "buildTool": "remix"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "15Gi", "ports": [{"container": 3000, "protocol": "http"}]}',
 'FROM node:18-alpine
WORKDIR /app
RUN npm install -g pnpm
EXPOSE 3000
CMD ["pnpm", "dev"]',
 'pnpm create remix@latest . --template remix-run/indie-stack',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- MERN Stack
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('mern-stack', 'MERN Stack', 'MongoDB + Express + React + Node.js 全栈开发', '全栈',
 ARRAY['MongoDB', 'Express', 'React', 'Node.js', 'Full Stack'], '🥞',
 '{"language": "javascript", "version": "18", "framework": "mern", "buildTool": "vite"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "20Gi", "ports": [{"container": 3000, "protocol": "http"}, {"container": 5000, "protocol": "http"}]}',
 'FROM node:18-alpine
WORKDIR /app
RUN npm install -g pnpm concurrently
EXPOSE 3000 5000
CMD ["pnpm", "dev"]',
 'mkdir -p client server && cd client && pnpm create vite . --template react-ts && cd ../server && pnpm init && pnpm add express mongoose cors dotenv',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- T3 Stack
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('t3-stack', 'T3 Stack', 'Next.js + tRPC + Prisma + Tailwind 类型安全全栈', '全栈',
 ARRAY['Next.js', 'tRPC', 'Prisma', 'Tailwind', 'TypeScript'], '🚀',
 '{"language": "typescript", "version": "18", "framework": "t3", "buildTool": "next"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "15Gi", "ports": [{"container": 3000, "protocol": "http"}]}',
 'FROM node:18-alpine
WORKDIR /app
RUN npm install -g pnpm
EXPOSE 3000
CMD ["pnpm", "dev"]',
 'pnpm create t3-app@latest . --noGit --CI',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- ============================================
-- BACKEND TEMPLATES (后端)
-- ============================================

-- Node.js 18 + Express
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('node-express', 'Node.js + Express', 'Node.js 18 + Express + TypeScript 后端开发环境', '后端',
 ARRAY['Node.js', 'Express', 'TypeScript', 'REST API'], '🟢',
 '{"language": "javascript", "version": "18", "framework": "express", "buildTool": "tsc"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "10Gi", "ports": [{"container": 3000, "protocol": "http"}]}',
 'FROM node:18-alpine
WORKDIR /app
RUN npm install -g pnpm nodemon ts-node
EXPOSE 3000
CMD ["pnpm", "dev"]',
 'pnpm init && pnpm add express cors helmet && pnpm add -D typescript @types/node @types/express nodemon ts-node',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- NestJS
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('nestjs', 'NestJS', 'NestJS + TypeScript 企业级 Node.js 框架', '后端',
 ARRAY['NestJS', 'TypeScript', 'Node.js', 'Enterprise', 'DI'], '🐱',
 '{"language": "typescript", "version": "18", "framework": "nestjs", "buildTool": "nest-cli"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "15Gi", "ports": [{"container": 3000, "protocol": "http"}]}',
 'FROM node:18-alpine
WORKDIR /app
RUN npm install -g pnpm @nestjs/cli
EXPOSE 3000
CMD ["pnpm", "start:dev"]',
 'nest new . --skip-git --package-manager pnpm',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Fastify
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('fastify', 'Fastify', 'Fastify + TypeScript 高性能 Node.js 框架', '后端',
 ARRAY['Fastify', 'TypeScript', 'Node.js', 'High Performance'], '⚡',
 '{"language": "typescript", "version": "18", "framework": "fastify", "buildTool": "tsc"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "10Gi", "ports": [{"container": 3000, "protocol": "http"}]}',
 'FROM node:18-alpine
WORKDIR /app
RUN npm install -g pnpm
EXPOSE 3000
CMD ["pnpm", "dev"]',
 'pnpm init && pnpm add fastify @fastify/cors && pnpm add -D typescript @types/node tsx',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Python FastAPI
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('python-fastapi', 'Python FastAPI', 'Python 3.11 + FastAPI 现代异步 API 框架', '后端',
 ARRAY['Python', 'FastAPI', 'Async', 'OpenAPI', 'Pydantic'], '🐍',
 '{"language": "python", "version": "3.11", "framework": "fastapi", "buildTool": "poetry"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "10Gi", "ports": [{"container": 8000, "protocol": "http"}]}',
 'FROM python:3.11-slim
WORKDIR /app
RUN pip install poetry
EXPOSE 8000
CMD ["poetry", "run", "uvicorn", "main:app", "--host", "0.0.0.0", "--reload"]',
 'poetry init -n && poetry add fastapi uvicorn[standard] pydantic',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Python Django
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('python-django', 'Python Django', 'Python 3.11 + Django 全功能 Web 框架', '后端',
 ARRAY['Python', 'Django', 'ORM', 'Admin', 'Full Featured'], '🎸',
 '{"language": "python", "version": "3.11", "framework": "django", "buildTool": "poetry"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "15Gi", "ports": [{"container": 8000, "protocol": "http"}]}',
 'FROM python:3.11-slim
WORKDIR /app
RUN pip install poetry
EXPOSE 8000
CMD ["poetry", "run", "python", "manage.py", "runserver", "0.0.0.0:8000"]',
 'poetry init -n && poetry add django djangorestframework && django-admin startproject app . ',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Python Flask
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('python-flask', 'Python Flask', 'Python 3.11 + Flask 轻量级 Web 框架', '后端',
 ARRAY['Python', 'Flask', 'Lightweight', 'Microframework'], '🧪',
 '{"language": "python", "version": "3.11", "framework": "flask", "buildTool": "poetry"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "10Gi", "ports": [{"container": 5000, "protocol": "http"}]}',
 'FROM python:3.11-slim
WORKDIR /app
RUN pip install poetry
EXPOSE 5000
CMD ["poetry", "run", "flask", "run", "--host=0.0.0.0"]',
 'poetry init -n && poetry add flask flask-cors python-dotenv',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Go Gin
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('go-gin', 'Go + Gin', 'Go 1.21 + Gin 高性能 Web 框架', '后端',
 ARRAY['Go', 'Gin', 'High Performance', 'REST API'], '🐹',
 '{"language": "go", "version": "1.21", "framework": "gin", "buildTool": "go"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "10Gi", "ports": [{"container": 8080, "protocol": "http"}]}',
 'FROM golang:1.21-alpine
WORKDIR /app
RUN go install github.com/cosmtrek/air@latest
EXPOSE 8080
CMD ["air"]',
 'go mod init app && go get -u github.com/gin-gonic/gin',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Go Fiber
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('go-fiber', 'Go + Fiber', 'Go 1.21 + Fiber Express 风格 Web 框架', '后端',
 ARRAY['Go', 'Fiber', 'Express-like', 'Fast'], '🚀',
 '{"language": "go", "version": "1.21", "framework": "fiber", "buildTool": "go"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "10Gi", "ports": [{"container": 3000, "protocol": "http"}]}',
 'FROM golang:1.21-alpine
WORKDIR /app
RUN go install github.com/cosmtrek/air@latest
EXPOSE 3000
CMD ["air"]',
 'go mod init app && go get -u github.com/gofiber/fiber/v2',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Go Echo
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('go-echo', 'Go + Echo', 'Go 1.21 + Echo 高性能极简 Web 框架', '后端',
 ARRAY['Go', 'Echo', 'Minimalist', 'High Performance'], '📢',
 '{"language": "go", "version": "1.21", "framework": "echo", "buildTool": "go"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "10Gi", "ports": [{"container": 8080, "protocol": "http"}]}',
 'FROM golang:1.21-alpine
WORKDIR /app
RUN go install github.com/cosmtrek/air@latest
EXPOSE 8080
CMD ["air"]',
 'go mod init app && go get -u github.com/labstack/echo/v4',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Rust Axum
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('rust-axum', 'Rust + Axum', 'Rust + Axum 类型安全高性能 Web 框架', '后端',
 ARRAY['Rust', 'Axum', 'Tokio', 'Type Safe', 'High Performance'], '🦀',
 '{"language": "rust", "version": "1.75", "framework": "axum", "buildTool": "cargo"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "20Gi", "ports": [{"container": 3000, "protocol": "http"}]}',
 'FROM rust:1.75-slim
WORKDIR /app
RUN cargo install cargo-watch
EXPOSE 3000
CMD ["cargo", "watch", "-x", "run"]',
 'cargo init && cargo add axum tokio --features tokio/full && cargo add serde --features derive',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Rust Actix
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('rust-actix', 'Rust + Actix', 'Rust + Actix Web 极速 Web 框架', '后端',
 ARRAY['Rust', 'Actix', 'Actor Model', 'Blazing Fast'], '🎭',
 '{"language": "rust", "version": "1.75", "framework": "actix", "buildTool": "cargo"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "20Gi", "ports": [{"container": 8080, "protocol": "http"}]}',
 'FROM rust:1.75-slim
WORKDIR /app
RUN cargo install cargo-watch
EXPOSE 8080
CMD ["cargo", "watch", "-x", "run"]',
 'cargo init && cargo add actix-web actix-rt && cargo add serde --features derive',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Java Spring Boot
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('java-spring', 'Java Spring Boot', 'Java 21 + Spring Boot 3 企业级框架', '后端',
 ARRAY['Java', 'Spring Boot', 'Enterprise', 'JPA', 'Security'], '☕',
 '{"language": "java", "version": "21", "framework": "spring-boot", "buildTool": "gradle"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "20Gi", "ports": [{"container": 8080, "protocol": "http"}]}',
 'FROM eclipse-temurin:21-jdk-alpine
WORKDIR /app
RUN apk add --no-cache gradle
EXPOSE 8080
CMD ["gradle", "bootRun"]',
 'curl https://start.spring.io/starter.tgz -d type=gradle-project -d language=java -d bootVersion=3.2.0 -d baseDir=. -d groupId=com.example -d artifactId=demo -d name=demo -d packageName=com.example.demo -d javaVersion=21 | tar -xzvf -',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Kotlin Ktor
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('kotlin-ktor', 'Kotlin + Ktor', 'Kotlin + Ktor 异步 Web 框架', '后端',
 ARRAY['Kotlin', 'Ktor', 'Coroutines', 'Async', 'JVM'], '🎯',
 '{"language": "kotlin", "version": "21", "framework": "ktor", "buildTool": "gradle"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "15Gi", "ports": [{"container": 8080, "protocol": "http"}]}',
 'FROM eclipse-temurin:21-jdk-alpine
WORKDIR /app
RUN apk add --no-cache gradle
EXPOSE 8080
CMD ["gradle", "run"]',
 'curl -o ktor.zip "https://start.ktor.io/p/ktor-sample?addSampleCode=true&engine=netty&configurationIn=YAML" && unzip ktor.zip && rm ktor.zip',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- ============================================
-- DATABASE TEMPLATES (数据库)
-- ============================================

-- PostgreSQL 16
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('postgres-16', 'PostgreSQL 16', 'PostgreSQL 16 关系型数据库开发环境', '数据库',
 ARRAY['PostgreSQL', 'SQL', 'ACID', 'Relational'], '🐘',
 '{"language": "sql", "version": "16", "framework": "postgresql", "buildTool": "psql"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "20Gi", "ports": [{"container": 5432, "protocol": "tcp"}]}',
 'FROM postgres:16-alpine
ENV POSTGRES_USER=devbox
ENV POSTGRES_PASSWORD=devbox
ENV POSTGRES_DB=devbox
EXPOSE 5432',
 NULL,
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- MySQL 8
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('mysql-8', 'MySQL 8', 'MySQL 8 关系型数据库开发环境', '数据库',
 ARRAY['MySQL', 'SQL', 'Relational', 'InnoDB'], '🐬',
 '{"language": "sql", "version": "8", "framework": "mysql", "buildTool": "mysql"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "20Gi", "ports": [{"container": 3306, "protocol": "tcp"}]}',
 'FROM mysql:8
ENV MYSQL_ROOT_PASSWORD=devbox
ENV MYSQL_DATABASE=devbox
ENV MYSQL_USER=devbox
ENV MYSQL_PASSWORD=devbox
EXPOSE 3306',
 NULL,
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- MongoDB 7
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('mongodb-7', 'MongoDB 7', 'MongoDB 7 文档数据库开发环境', '数据库',
 ARRAY['MongoDB', 'NoSQL', 'Document', 'JSON'], '🍃',
 '{"language": "nosql", "version": "7", "framework": "mongodb", "buildTool": "mongosh"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "20Gi", "ports": [{"container": 27017, "protocol": "tcp"}]}',
 'FROM mongo:7
ENV MONGO_INITDB_ROOT_USERNAME=devbox
ENV MONGO_INITDB_ROOT_PASSWORD=devbox
EXPOSE 27017',
 NULL,
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Redis 7
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('redis-7', 'Redis 7', 'Redis 7 内存数据库和缓存', '数据库',
 ARRAY['Redis', 'Cache', 'In-Memory', 'Key-Value'], '🔴',
 '{"language": "nosql", "version": "7", "framework": "redis", "buildTool": "redis-cli"}',
 '{"cpu": "0.5", "memory": "1Gi", "storage": "5Gi", "ports": [{"container": 6379, "protocol": "tcp"}]}',
 'FROM redis:7-alpine
EXPOSE 6379
CMD ["redis-server", "--appendonly", "yes"]',
 NULL,
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Elasticsearch 8
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('elasticsearch-8', 'Elasticsearch 8', 'Elasticsearch 8 搜索和分析引擎', '数据库',
 ARRAY['Elasticsearch', 'Search', 'Analytics', 'Full-text'], '🔍',
 '{"language": "nosql", "version": "8", "framework": "elasticsearch", "buildTool": "curl"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "30Gi", "ports": [{"container": 9200, "protocol": "http"}, {"container": 9300, "protocol": "tcp"}]}',
 'FROM elasticsearch:8.11.0
ENV discovery.type=single-node
ENV xpack.security.enabled=false
EXPOSE 9200 9300',
 NULL,
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- ClickHouse
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('clickhouse', 'ClickHouse', 'ClickHouse 列式 OLAP 数据库', '数据库',
 ARRAY['ClickHouse', 'OLAP', 'Analytics', 'Columnar'], '🏠',
 '{"language": "sql", "version": "23", "framework": "clickhouse", "buildTool": "clickhouse-client"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "30Gi", "ports": [{"container": 8123, "protocol": "http"}, {"container": 9000, "protocol": "tcp"}]}',
 'FROM clickhouse/clickhouse-server:23
EXPOSE 8123 9000',
 NULL,
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- ============================================
-- DEVOPS TEMPLATES (DevOps)
-- ============================================

-- Docker Development
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('docker-dev', 'Docker Development', 'Docker 开发和容器化环境', 'DevOps',
 ARRAY['Docker', 'Container', 'DevOps', 'CI/CD'], '🐳',
 '{"language": "yaml", "version": "24", "framework": "docker", "buildTool": "docker"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "30Gi", "ports": []}',
 'FROM docker:24-dind
RUN apk add --no-cache docker-compose curl git
EXPOSE 2375 2376',
 NULL,
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Kubernetes Development
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('k8s-dev', 'Kubernetes Development', 'Kubernetes 开发和部署环境', 'DevOps',
 ARRAY['Kubernetes', 'K8s', 'Container Orchestration', 'Helm'], '☸️',
 '{"language": "yaml", "version": "1.28", "framework": "kubernetes", "buildTool": "kubectl"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "20Gi", "ports": []}',
 'FROM alpine:3.18
RUN apk add --no-cache curl bash git
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && chmod +x kubectl && mv kubectl /usr/local/bin/
RUN curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash',
 NULL,
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Terraform
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('terraform', 'Terraform', 'Terraform 基础设施即代码环境', 'DevOps',
 ARRAY['Terraform', 'IaC', 'Infrastructure', 'Cloud'], '🏗️',
 '{"language": "hcl", "version": "1.6", "framework": "terraform", "buildTool": "terraform"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "10Gi", "ports": []}',
 'FROM hashicorp/terraform:1.6
RUN apk add --no-cache git curl bash
WORKDIR /app',
 NULL,
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Ansible
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('ansible', 'Ansible', 'Ansible 自动化运维环境', 'DevOps',
 ARRAY['Ansible', 'Automation', 'Configuration Management', 'Playbook'], '🔧',
 '{"language": "yaml", "version": "2.15", "framework": "ansible", "buildTool": "ansible-playbook"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "10Gi", "ports": []}',
 'FROM python:3.11-slim
RUN pip install ansible ansible-lint
WORKDIR /app',
 NULL,
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- ============================================
-- OTHER TEMPLATES (其他)
-- ============================================

-- Data Science Python
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('data-science', 'Data Science Python', 'Python 数据科学和机器学习环境', '其他',
 ARRAY['Python', 'Data Science', 'Machine Learning', 'Jupyter', 'Pandas'], '📊',
 '{"language": "python", "version": "3.11", "framework": "jupyter", "buildTool": "pip"}',
 '{"cpu": "4", "memory": "8Gi", "storage": "50Gi", "ports": [{"container": 8888, "protocol": "http"}]}',
 'FROM python:3.11-slim
RUN pip install jupyter pandas numpy matplotlib scikit-learn seaborn
WORKDIR /app
EXPOSE 8888
CMD ["jupyter", "notebook", "--ip=0.0.0.0", "--allow-root", "--no-browser"]',
 NULL,
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Deno
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('deno', 'Deno', 'Deno 安全的 JavaScript/TypeScript 运行时', '其他',
 ARRAY['Deno', 'TypeScript', 'JavaScript', 'Secure', 'Modern'], '🦕',
 '{"language": "typescript", "version": "1.38", "framework": "deno", "buildTool": "deno"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "10Gi", "ports": [{"container": 8000, "protocol": "http"}]}',
 'FROM denoland/deno:1.38.0
WORKDIR /app
EXPOSE 8000
CMD ["deno", "run", "--allow-net", "--allow-read", "--watch", "main.ts"]',
 'echo ''import { serve } from "https://deno.land/std@0.208.0/http/server.ts";\nserve((req) => new Response("Hello World!"), { port: 8000 });'' > main.ts',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Bun
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('bun', 'Bun', 'Bun 超快 JavaScript 运行时和工具链', '其他',
 ARRAY['Bun', 'JavaScript', 'TypeScript', 'Fast', 'All-in-one'], '🥟',
 '{"language": "javascript", "version": "1.0", "framework": "bun", "buildTool": "bun"}',
 '{"cpu": "1", "memory": "2Gi", "storage": "10Gi", "ports": [{"container": 3000, "protocol": "http"}]}',
 'FROM oven/bun:1
WORKDIR /app
EXPOSE 3000
CMD ["bun", "run", "--watch", "index.ts"]',
 'bun init -y',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- Elixir Phoenix
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('elixir-phoenix', 'Elixir Phoenix', 'Elixir + Phoenix 实时 Web 框架', '其他',
 ARRAY['Elixir', 'Phoenix', 'Functional', 'Real-time', 'BEAM'], '🔥',
 '{"language": "elixir", "version": "1.15", "framework": "phoenix", "buildTool": "mix"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "15Gi", "ports": [{"container": 4000, "protocol": "http"}]}',
 'FROM elixir:1.15-alpine
RUN mix local.hex --force && mix local.rebar --force && mix archive.install hex phx_new --force
WORKDIR /app
EXPOSE 4000
CMD ["mix", "phx.server"]',
 'mix phx.new . --app app --no-ecto --no-mailer --no-gettext --no-dashboard',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();

-- C/C++ Development
INSERT INTO templates (name, display_name, description, category, tags, icon, runtime, default_config, dockerfile, init_script, created_by)
VALUES
('cpp-dev', 'C/C++ Development', 'C/C++ 系统编程开发环境', '其他',
 ARRAY['C', 'C++', 'Systems Programming', 'CMake', 'GCC'], '⚙️',
 '{"language": "cpp", "version": "13", "framework": "cmake", "buildTool": "cmake"}',
 '{"cpu": "2", "memory": "4Gi", "storage": "20Gi", "ports": []}',
 'FROM gcc:13
RUN apt-get update && apt-get install -y cmake gdb valgrind clang-format
WORKDIR /app',
 'mkdir -p build && echo ''cmake_minimum_required(VERSION 3.20)\nproject(app)\nadd_executable(app main.cpp)'' > CMakeLists.txt && echo ''#include <iostream>\nint main() { std::cout << "Hello World!" << std::endl; return 0; }'' > main.cpp',
 '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO UPDATE SET updated_at = NOW();
