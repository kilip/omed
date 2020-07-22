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

namespace Omed\Laravel\Security;

use Illuminate\Foundation\Application;
use Illuminate\Support\ServiceProvider;
use Kilip\SanctumORM\Contracts\TokenModelInterface;
use Omed\Laravel\Security\Http\Controllers\AuthController;
use Omed\Laravel\Security\Model\Tokens;

class SecurityServiceProvider extends ServiceProvider
{
    public function boot(Application $app)
    {
        $this->publishes([
            __DIR__.'/Resources/config/security.php' => config_path('omed/security.php'),
        ], 'config');
        $app->alias(AuthController::class, 'OmedSecurityAuthController');

        $app['config']->set('auth.guards.api.driver', 'sanctum');
        $this->loadRoutesFrom(__DIR__.'/Resources/routes/api.php');
    }

    public function register()
    {
        $this->configureModel();
    }

    private function configureModel()
    {
        $mappings = [
            __NAMESPACE__.'\\Model' => [
                'type' => 'annotation',
                'dir' => __DIR__.'/Model',
            ],
        ];
        $key = 'doctrine.managers.'.config('omed.security.entity_manager_name', 'default').'.mappings';
        $mappings = array_merge($mappings, config($key, []));
        config([
            $key => $mappings,
        ]);

        // configure target entities
        $resolves = config('doctrine.resolve_target_entities', []);
        $resolves = array_merge([
            TokenModelInterface::class => Tokens::class,
        ], $resolves);

        config([
            'doctrine.resolve_target_entities' => array_merge(
                $resolves,
                config('doctrine.resolve_target_entities', [])
            ),
            'sanctum.orm.models.token' => Tokens::class,
        ]);
    }
}
