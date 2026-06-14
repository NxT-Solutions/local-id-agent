<?php

use App\Http\Controllers\LocalIdController;
use Illuminate\Support\Facades\Route;

Route::post('/localid/challenge', [LocalIdController::class, 'challenge']);
Route::post('/localid/verify', [LocalIdController::class, 'verify']);
