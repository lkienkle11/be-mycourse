// PM2 cannot natively "swap back" a broken Go binary. CI backs up bin/mycourse-io-be-dev -> .prev then rsyncs the new build
// over the same path; scripts/pm2-reload-with-binary-rollback.sh backs up ecosystem.config.cjs, pulls ecosystem from git, reloads, health-checks, and rolls back both if needed.
// max_restarts + min_uptime: if the process exits before min_uptime too many times, PM2 stops autorestarting (same policy for all apps).
const SHARED_DEPLOY_PATH = process.env.DEPLOY_PATH;
const DEPLOY_PATH_DEV = process.env.DEPLOY_PATH_DEV || SHARED_DEPLOY_PATH || '/var/www/be-mycourse';
const DEPLOY_PATH_STG = process.env.DEPLOY_PATH_STG || SHARED_DEPLOY_PATH || '/var/www/be-mycourse';
const DEPLOY_PATH_MAIN = process.env.DEPLOY_PATH_MAIN || SHARED_DEPLOY_PATH || '/var/www/be-mycourse';

const DEPLOY_ENV_FILE_DEV = process.env.DEPLOY_ENV_FILE_DEV || `${DEPLOY_PATH_DEV}/.env`;
const DEPLOY_ENV_FILE_STG = process.env.DEPLOY_ENV_FILE_STG || `${DEPLOY_PATH_STG}/.env.staging`;
const DEPLOY_ENV_FILE_MAIN = process.env.DEPLOY_ENV_FILE_MAIN || `${DEPLOY_PATH_MAIN}/.env.prod`;

module.exports = {
    apps: [
        {
            name: 'mycourse-api-dev',
            cwd: DEPLOY_PATH_DEV,           // PM2 sẽ "cd" vào thư mục này trước khi chạy
            script: './bin/mycourse-io-be-dev',        // PM2 tìm file này bên trong cwd ở trên
            instances: 1,
            autorestart: true,
            min_uptime: '5s',
            max_restarts: 3,
            max_memory_restart: '1024M',
            env: {
                CGO_ENABLED: "1",
                MIGRATE: "1",
            },
            env_file: DEPLOY_ENV_FILE_DEV, // Khai báo đường dẫn tuyệt đối tới file .env
        },
        {
            name: 'mycourse-api-staging',
            cwd: DEPLOY_PATH_STG,           // PM2 sẽ "cd" vào thư mục này trước khi chạy
            script: './bin/mycourse-io-be-staging',        // PM2 tìm file này bên trong cwd ở trên
            instances: 1,
            autorestart: true,
            min_uptime: '5s',
            max_restarts: 3,
            max_memory_restart: '1024M',
            env: {
                STAGE: 'staging',
                CGO_ENABLED: "1",
                MIGRATE: "1",
            },
            env_file: DEPLOY_ENV_FILE_STG, // Khai báo đường dẫn tuyệt đối tới file .env
        },
        {
            name: 'mycourse-api-prod',
            cwd: DEPLOY_PATH_MAIN,           // PM2 sẽ "cd" vào thư mục này trước khi chạy
            script: './bin/mycourse-io-be-prod',        // PM2 tìm file này bên trong cwd ở trên
            instances: 1,
            autorestart: true,
            min_uptime: '5s',
            max_restarts: 3,
            max_memory_restart: '1024M',
            env: {
                STAGE: 'prod',
                CGO_ENABLED: "1",
                MIGRATE: "1",
            },
            env_file: DEPLOY_ENV_FILE_MAIN, // Khai báo đường dẫn tuyệt đối tới file .env
        }
    ],
};
