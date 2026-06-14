<?php

namespace App\Services;

use DateTimeImmutable;
use DateTimeInterface;
use Exception;
use Illuminate\Http\Exceptions\HttpResponseException;
use Illuminate\Support\Facades\Response;

class LocalIdVerifier
{
    private const CHALLENGE_MAX_AGE_SECONDS = 60;

    private const EXPECTED_BACKEND = 'http://localhost:8000';

    /** @var list<string> */
    private const ALLOWED_ORIGINS = [
        'http://localhost:5173',
        'http://localhost:5174',
    ];

    /**
     * @param array<string, string> $body
     * @return array{success: bool, user: array{id: string, name: string}}
     */
    public function verify(array $body): array
    {
        $algorithm = $body['algorithm'] ?? '';
        $purpose = $body['purpose'] ?? '';
        $backend = $body['backend'] ?? '';
        $origin = $body['origin'] ?? '';
        $certificate = $body['certificate'] ?? '';
        $signature = $body['signature'] ?? '';
        $signedAt = $body['signedAt'] ?? '';
        $challenge = $body['challenge'] ?? '';

        if ($algorithm !== 'RS256') {
            $this->deny('unsupported algorithm');
        }

        if ($purpose !== 'login') {
            $this->deny('purpose is not allowed');
        }

        if ($backend !== self::EXPECTED_BACKEND) {
            $this->deny('backend is not allowed');
        }

        if (! in_array($origin, self::ALLOWED_ORIGINS, true)) {
            $this->deny('origin is not allowed');
        }

        if ($certificate === '') {
            $this->badRequest('certificate is required');
        }

        try {
            $signedTime = new DateTimeImmutable($signedAt);
        } catch (Exception) {
            $this->badRequest('signedAt must be RFC3339');
        }

        $now = new DateTimeImmutable('now', new \DateTimeZone('UTC'));
        $age = $now->getTimestamp() - $signedTime->setTimezone(new \DateTimeZone('UTC'))->getTimestamp();

        if ($age < 0 || $age > self::CHALLENGE_MAX_AGE_SECONDS) {
            $this->deny('challenge timestamp is stale or invalid');
        }

        $payload = $this->buildCanonicalPayload(
            backend: $backend,
            challenge: $challenge,
            origin: $origin,
            purpose: $purpose,
            timestamp: $signedAt,
        );

        $publicKey = openssl_pkey_get_public($this->certificatePem($certificate));
        if ($publicKey === false) {
            $this->deny('invalid certificate');
        }

        $signatureBytes = $this->decodeBase64Url($signature);
        if ($signatureBytes === null) {
            $this->badRequest('signature must be valid base64url');
        }

        $verified = openssl_verify($payload, $signatureBytes, $publicKey, OPENSSL_ALGO_SHA256);
        if ($verified !== 1) {
            $this->deny('signature verification failed');
        }

        return [
            'success' => true,
            'user' => [
                'id' => 'mock-user',
                'name' => 'Mock Dev User',
            ],
        ];
    }

    private function buildCanonicalPayload(
        string $backend,
        string $challenge,
        string $origin,
        string $purpose,
        string $timestamp,
    ): string {
        if ($challenge === '' || ! preg_match('/^[A-Za-z0-9_-]+$/', $challenge)) {
            $this->badRequest('challenge must be valid base64url');
        }

        $payload = [
            'backend' => $backend,
            'challenge' => $challenge,
            'origin' => $origin,
            'purpose' => $purpose,
            'timestamp' => $timestamp,
        ];

        return json_encode($payload, JSON_UNESCAPED_SLASHES | JSON_THROW_ON_ERROR);
    }

    private function certificatePem(string $certificateB64): string
    {
        $der = base64_decode($certificateB64, true);
        if ($der === false) {
            $this->deny('certificate must be valid base64');
        }

        $pem = "-----BEGIN CERTIFICATE-----\n";
        $pem .= chunk_split(base64_encode($der), 64, "\n");
        $pem .= "-----END CERTIFICATE-----\n";

        return $pem;
    }

    private function decodeBase64Url(string $value): ?string
    {
        if ($value === '' || ! preg_match('/^[A-Za-z0-9_-]+$/', $value)) {
            return null;
        }

        $padded = strtr($value, '-_', '+/');
        $padded .= str_repeat('=', (4 - strlen($padded) % 4) % 4);

        $decoded = base64_decode($padded, true);

        return $decoded === false ? null : $decoded;
    }

    private function deny(string $message): never
    {
        throw new HttpResponseException(
            Response::json(['error' => $message], 403)
        );
    }

    private function badRequest(string $message): never
    {
        throw new HttpResponseException(
            Response::json(['error' => $message], 400)
        );
    }
}
