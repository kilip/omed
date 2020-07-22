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

namespace Omed\Laravel\Security\Tests;

use Kilip\LaravelDoctrine\ORM\KilipDoctrineServiceProvider;
use Kilip\SanctumORM\SanctumORMServiceProvider;
use Laravel\Sanctum\SanctumServiceProvider;
use LaravelDoctrine\Extensions\GedmoExtensionsServiceProvider;
use LaravelDoctrine\ORM\DoctrineServiceProvider;
use Omed\Laravel\Core\CoreServiceProvider;
use Omed\Laravel\Security\Model\Tokens;
use Omed\Laravel\Security\SecurityServiceProvider;
use Omed\Laravel\Security\Tests\Resources\Model\TestSecurityUser;
use Omed\Laravel\Security\Tests\Resources\TestHttpKernel;
use Orchestra\Testbench\TestCase as BaseTestCase;

class TestCase extends BaseTestCase
{
    protected function setUp(): void
    {
        parent::setUp();
        $this->artisan('doctrine:schema:update', ['--force' => true]);
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
        ];
    }

    protected function getEnvironmentSetUp($app)
    {
        parent::getEnvironmentSetUp($app);

        /** @var \Illuminate\Config\Repository $config */
        $config = $app['config'];

        $config->set('doctrine.managers.default.paths', [__DIR__.'/Resources/Model']);

        $config->set('auth.providers.users.driver', 'doctrine');
        $config->set('auth.providers.users.model', TestSecurityUser::class);
        $config->set('sanctum.orm.models.token', Tokens::class);
        $config->set('sanctum.orm.models.user', TestSecurityUser::class);
    }

    protected function resolveApplicationHttpKernel($app)
    {
        $app->singleton(
            'Illuminate\\Contracts\\Http\\Kernel',
            TestHttpKernel::class
        );
    }
}
