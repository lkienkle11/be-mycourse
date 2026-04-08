module.exports = {
    apps: [
        {
            name: 'mycourse-api-dev',
            cwd: '/var/www/be-mycourse',           // PM2 sẽ "cd" vào thư mục này trước khi chạy
            script: './bin/mycourse-io-be-dev',        // PM2 tìm file này bên trong cwd ở trên
            instances: 1,
            autorestart: true,
            max_memory_restart: '1024M',
            env: {},
            env_file: '/var/www/be-mycourse/.env', // Khai báo đường dẫn tuyệt đối tới file .env
        },
        {
            name: 'mycourse-api-staging',
            cwd: '/var/www/be-mycourse',           // PM2 sẽ "cd" vào thư mục này trước khi chạy
            script: './bin/mycourse-io-be-staging',        // PM2 tìm file này bên trong cwd ở trên
            instances: 1,
            autorestart: true,
            max_memory_restart: '1024M',
            env: {
                STAGE: 'staging',
            },
            env_file: '/var/www/be-mycourse/.env.staging', // Khai báo đường dẫn tuyệt đối tới file .env
        },
        {
            name: 'mycourse-api-prod',
            cwd: '/var/www/be-mycourse',           // PM2 sẽ "cd" vào thư mục này trước khi chạy
            script: './bin/mycourse-io-be-prod',        // PM2 tìm file này bên trong cwd ở trên
            instances: 1,
            autorestart: true,
            max_memory_restart: '1024M',
            env: {
                STAGE: 'prod',
            },
            env_file: '/var/www/be-mycourse/.env.prod', // Khai báo đường dẫn tuyệt đối tới file .env
        }
    ],
};
