<?php

namespace App\Http\Controllers;

use App\Services\LocalIdVerifier;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Str;

class LocalIdController extends Controller
{
    public function __construct(private readonly LocalIdVerifier $verifier) {}

    public function challenge(): JsonResponse
    {
        $challenge = rtrim(strtr(base64_encode(random_bytes(32)), '+/', '-_'), '=');
        Cache::put($this->cacheKey($challenge), true, now()->addSeconds(60));

        return response()->json(['challenge' => $challenge]);
    }

    public function verify(Request $request): JsonResponse
    {
        $body = $request->json()->all();
        $challenge = (string) ($body['challenge'] ?? '');

        if ($challenge === '' || ! Cache::pull($this->cacheKey($challenge))) {
            return response()->json(['error' => 'challenge not found or already used'], 403);
        }

        return response()->json($this->verifier->verify($body));
    }

    private function cacheKey(string $challenge): string
    {
        return 'localid:challenge:'.hash('sha256', $challenge);
    }
}
