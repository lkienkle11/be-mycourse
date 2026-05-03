// PM2 cannot natively "swap back" a broken Go binary. CI backs up bin/mycourse-io-be-dev -> .prev then rsyncs the new build
// over the same path; scripts/pm2-reload-with-binary-rollback.sh backs up ecosystem.config.cjs, pulls ecosystem from git, reloads, health-checks, and rolls back both if needed.
// max_restarts + min_uptime: if the process exits before min_uptime too many times, PM2 stops autorestarting (same policy for all apps).
module.exports = {
    apps: [
        {
            name: 'mycourse-api-dev',
            cwd: '/var/www/be-mycourse',           // PM2 sẽ "cd" vào thư mục này trước khi chạy
            script: './bin/mycourse-io-be-dev',        // PM2 tìm file này bên trong cwd ở trên
            instances: 1,
            autorestart: true,
            min_uptime: '5s',
            max_restarts: 3,
            max_memory_restart: '1024M',
            env: {
                CGO_ENABLED: "1",
            },
            env_file: '/var/www/be-mycourse/.env', // Khai báo đường dẫn tuyệt đối tới file .env
        },
        {
            name: 'mycourse-api-staging',
            cwd: '/var/www/be-mycourse',           // PM2 sẽ "cd" vào thư mục này trước khi chạy
            script: './bin/mycourse-io-be-staging',        // PM2 tìm file này bên trong cwd ở trên
            instances: 1,
            autorestart: true,
            min_uptime: '5s',
            max_restarts: 3,
            max_memory_restart: '1024M',
            env: {
                STAGE: 'staging',
                CGO_ENABLED: "1",
            },
            env_file: '/var/www/be-mycourse/.env.staging', // Khai báo đường dẫn tuyệt đối tới file .env
        },
        {
            name: 'mycourse-api-prod',
            cwd: '/var/www/be-mycourse',           // PM2 sẽ "cd" vào thư mục này trước khi chạy
            script: './bin/mycourse-io-be-prod',        // PM2 tìm file này bên trong cwd ở trên
            instances: 1,
            autorestart: true,
            min_uptime: '5s',
            max_restarts: 3,
            max_memory_restart: '1024M',
            env: {
                STAGE: 'prod',
                CGO_ENABLED: "1",
            },
            env_file: '/var/www/be-mycourse/.env.prod', // Khai báo đường dẫn tuyệt đối tới file .env
        }
    ],
};
