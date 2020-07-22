<?php

/*
 * This file is part of the Omed project.
 *
 * (c) Anthonius Munthi <https://itstoni.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

declare(strict_types=1);

namespace Omed\Laravel\User\Tests\Controllers;

use Omed\Laravel\User\Testing\UserManagerTrait;
use Omed\Laravel\User\Tests\UserTestCase;

class UserControllerTest extends UserTestCase
{
    use UserManagerTrait;

    public function testUnauthenticatedUserAccess()
    {
        $response = $this->json('GET', route('omed.routes.user.index'));
        $response->assertStatus(401);
    }

    public function testIndex()
    {
        $this->assertStringContainsString('/api/users', route('omed.routes.user.index'));
        $user = $this->generateUserData();
        $token = $this->createToken($user);
        $response = $this->json('GET', route('omed.routes.user.index'), [], [
            'Authorization' => 'Bearer '.$token->plainTextToken,
        ]);
        $response->assertStatus(200);
    }

    public function testShow()
    {
        $user = $this->generateUserData();
        $token = $this->createToken($user);
        $this->assertStringContainsString('/api/users', route('omed.routes.user.show', ['user' => $user->getId()]));

        $response = $this->json('GET', route('omed.routes.user.show', ['user' => $user->getId()]), [], [
            'Authorization' => 'Bearer '.$token->plainTextToken,
        ]);

        $response->assertStatus(200);

        $this->assertEquals($user->getUsername(), $response->json('data.username'));
        $this->assertEquals($user->getEmail(), $response->json('data.email'));
    }
}
