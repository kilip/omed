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

namespace Omed\Laravel\User\Tests\Services;

use Doctrine\Persistence\ObjectManager;
use Illuminate\Http\Request;
use Omed\Laravel\User\Services\UserManager;
use Omed\Laravel\User\Tests\UserTestCase;

class UserManagerTest extends UserTestCase
{
    public function testConstruct()
    {
        $manager = $this->app->get(UserManager::class);
        $this->assertInstanceOf(UserManager::class, $manager);
    }

    public function testGetUserList()
    {
        $user = $this->generateUserData();
        $token = $this->createToken($user);
        $this->withToken($token->plainTextToken);

        $response = $this->get(route('omed.routes.user.index'));

        $response->assertStatus(200);
    }
}
