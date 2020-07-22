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

namespace Omed\Laravel\User\Tests;

use Kilip\LaravelDoctrine\ORM\KilipDoctrineServiceProvider;
use Kilip\LaravelDoctrine\ORM\Testing\ORMTestTrait;
use Kilip\SanctumORM\SanctumORMServiceProvider;
use Laravel\Sanctum\SanctumServiceProvider;
use LaravelDoctrine\ORM\DoctrineServiceProvider;
use Omed\Component\User\Model\UserInterface;
use Omed\Laravel\Core\CoreServiceProvider;
use Omed\Laravel\Security\SecurityServiceProvider;
use Omed\Laravel\User\Model\User;
use Omed\Laravel\User\SecurityEventServiceProvider;
use Omed\Laravel\User\Testing\UserManagerTrait;
use Omed\Laravel\User\UserServiceProvider;
use Orchestra\Testbench\TestCase as OrchestraTestCase;

class UserTestCase extends OrchestraTestCase
{
    use UserManagerTrait;
    use ORMTestTrait;

    protected function setUp(): void
    {
        parent::setUp();
        $this->recreateDatabase();
    }

    protected function getPackageProviders($app)
    {
        return [
            DoctrineServiceProvider::class,
            KilipDoctrineServiceProvider::class,
            SanctumServiceProvider::class,
            SanctumORMServiceProvider::class,
            CoreServiceProvider::class,
            SecurityServiceProvider::class,
            UserServiceProvider::class,
            SecurityEventServiceProvider::class,
        ];
    }

    protected function getPackageAliases($app)
    {
        return [
        ];
    }

    protected function getEnvironmentSetUp($app): void
    {
        $app['config']->set('database.default', 'sqlite');
        $app['config']->set('database.connections.sqlite', [
            'driver' => 'sqlite',
            'database' => ':memory:',
            'prefix' => '',
        ]);

        $defPath = sys_get_temp_dir().'/omed-user/foo-model';
        if (!is_dir($defPath)) {
            mkdir($defPath, 0777, true);
        }
        $app['config']->set('doctrine.managers.default.connection', 'sqlite');
        $app['config']->set('doctrine.managers.default.paths', [$defPath]);
        $app['config']->set('alice.doctrine_orm.default.paths', [
            __DIR__.'/Resources/fixtures',
        ]);
        $app['config']->set('alice.doctrine_orm.default.manager', 'default');
        $app['config']->set('sanctum.orm.models.user', User::class);
    }

    /**
     * @throws \Illuminate\Contracts\Container\BindingResolutionException
     *
     * @return object|UserInterface|null
     */
    protected function generateUserData()
    {
        $manager = $this->getUserManager();
        $user = $manager->findByUsername('test');
        if (null === $user) {
            $user = $manager->createUser();
            $user
                ->setUsername('test')
                ->setPlainPassword('test')
                ->setEmail('test@example.com');
            $manager->storeUser($user);
        }

        return $user;
    }
}
