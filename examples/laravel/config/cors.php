<?php

return [
    'paths' => ['localid/*', 'up'],
    'allowed_methods' => ['POST', 'OPTIONS', 'GET'],
    'allowed_origins' => [
        'http://localhost:5173',
        'http://localhost:5174',
    ],
    'allowed_origins_patterns' => [],
    'allowed_headers' => ['Content-Type'],
    'exposed_headers' => [],
    'max_age' => 0,
    'supports_credentials' => false,
];
